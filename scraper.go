package main

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// ScrapeResult holds the parsed information from a webpage
type ScrapeResult struct {
	Title string
	M3U8s []string
}

// ScrapePage fetches the URL and extracts the title and any .m3u8 URLs found.
func ScrapePage(pageURL string) (*ScrapeResult, error) {
	parsedURL, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("invalid page URL: %w", err)
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set common headers to mimic a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code from server: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	htmlContent := string(bodyBytes)
	title := extractTitle(htmlContent)
	m3u8s := extractM3U8s(htmlContent, parsedURL)

	return &ScrapeResult{
		Title: title,
		M3U8s: m3u8s,
	}, nil
}

func extractTitle(htmlContent string) string {
	// 1. Try og:title meta tag first (often cleaner than document title)
	ogTitleRegex := regexp.MustCompile(`(?i)<meta\s+property=["']og:title["']\s+content=["'](.*?)["']`)
	matches := ogTitleRegex.FindStringSubmatch(htmlContent)
	if len(matches) > 1 {
		return cleanTitle(matches[1])
	}

	// 2. Try standard <title> tag
	titleRegex := regexp.MustCompile(`(?i)<title[^>]*>(.*?)</title>`)
	matches = titleRegex.FindStringSubmatch(htmlContent)
	if len(matches) > 1 {
		return cleanTitle(matches[1])
	}

	// 3. Try twitter:title meta tag
	twitterTitleRegex := regexp.MustCompile(`(?i)<meta\s+name=["']twitter:title["']\s+content=["'](.*?)["']`)
	matches = twitterTitleRegex.FindStringSubmatch(htmlContent)
	if len(matches) > 1 {
		return cleanTitle(matches[1])
	}

	return ""
}

func cleanTitle(title string) string {
	title = html.UnescapeString(title)
	title = strings.TrimSpace(title)
	return title
}

func extractM3U8s(htmlContent string, baseURL *url.URL) []string {
	// Replace escaped slashes \/ with / (common in JSON/JS payloads)
	normalized := strings.ReplaceAll(htmlContent, `\/`, `/`)

	// 1. Match absolute URLs ending with .m3u8 plus optional query params
	absRe := regexp.MustCompile(`(?i)https?://[^"'\s><()]+?\.m3u8[^"'\s><()]*`)
	matches := absRe.FindAllString(normalized, -1)

	// 2. Match relative URLs inside quotes that contain .m3u8
	relRe := regexp.MustCompile(`(?i)["']([^"'\s><()]+?\.m3u8[^"'\s><()]*?)["']`)
	relMatches := relRe.FindAllStringSubmatch(normalized, -1)

	uniqueMap := make(map[string]bool)
	var result []string

	addURL := func(rawURL string) {
		// Clean html entities if present (e.g. &amp; in URLs)
		rawURL = html.UnescapeString(rawURL)
		
		u, err := url.Parse(rawURL)
		if err != nil {
			return
		}
		
		resolved := baseURL.ResolveReference(u).String()
		if !uniqueMap[resolved] {
			uniqueMap[resolved] = true
			result = append(result, resolved)
		}
	}

	for _, match := range matches {
		addURL(match)
	}

	for _, submatch := range relMatches {
		if len(submatch) > 1 {
			addURL(submatch[1])
		}
	}

	return result
}

// ScrapePageDynamic runs a headless browser, intercepts network requests to find .m3u8 URLs,
// and extracts the page title.
func ScrapePageDynamic(pageURL string) (*ScrapeResult, error) {
	parsedURL, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("invalid page URL: %w", err)
	}

	// Create allocator context with default options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox,
	)
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	// Create chromedp context
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set a timeout for dynamic loading
	ctx, cancel = context.WithTimeout(ctx, 35*time.Second)
	defer cancel()

	var m3u8s []string
	var title string
	uniqueMap := make(map[string]bool)
	var mu sync.Mutex

	// Listen for network requests
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			reqURL := e.Request.URL
			reqURL = strings.ReplaceAll(reqURL, `\/`, `/`)
			reqURL = html.UnescapeString(reqURL)

			if strings.Contains(strings.ToLower(reqURL), ".m3u8") {
				mu.Lock()
				u, err := url.Parse(reqURL)
				if err == nil {
					resolved := parsedURL.ResolveReference(u).String()
					if !uniqueMap[resolved] {
						uniqueMap[resolved] = true
						m3u8s = append(m3u8s, resolved)
					}
				}
				mu.Unlock()
			}
		}
	})

	fmt.Println("Opening headless browser to monitor network requests (waiting 10 seconds for streams to load)...")

	// Run tasks
	err = chromedp.Run(ctx,
		network.Enable(),
		chromedp.Navigate(pageURL),
		chromedp.Evaluate(`document.querySelector('meta[property="og:title"]')?.getAttribute('content') || document.title`, &title),
		chromedp.Sleep(10*time.Second),
	)
	if err != nil {
		mu.Lock()
		foundCount := len(m3u8s)
		mu.Unlock()
		// Only return error if we actually found no streams
		if foundCount == 0 {
			return nil, fmt.Errorf("headless browser failed: %w", err)
		}
	}

	title = cleanTitle(title)

	return &ScrapeResult{
		Title: title,
		M3U8s: m3u8s,
	}, nil
}

// ScrapeLinks extracts all links from the homepage.
// If selector is empty, it attempts static link extraction first and falls back to dynamic.
// If selector is specified, it uses the dynamic scraper directly.
func ScrapeLinks(pageURL string, selector string) ([]string, error) {
	if selector == "" {
		parsedURL, err := url.Parse(pageURL)
		if err == nil {
			htmlContent, err := fetchStaticHTML(pageURL)
			if err == nil {
				links := extractLinksFromHTML(htmlContent, parsedURL)
				if len(links) > 0 {
					return links, nil
				}
			}
		}
	}
	return ScrapeLinksDynamic(pageURL, selector)
}

func fetchStaticHTML(pageURL string) (string, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status code: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func extractLinksFromHTML(htmlContent string, baseURL *url.URL) []string {
	hrefRe := regexp.MustCompile(`(?i)<a\s+[^>]*href=["']([^"']+)["']`)
	matches := hrefRe.FindAllStringSubmatch(htmlContent, -1)
	
	var links []string
	for _, match := range matches {
		if len(match) > 1 {
			rawURL := strings.TrimSpace(match[1])
			if rawURL == "" || strings.HasPrefix(rawURL, "javascript:") || strings.HasPrefix(rawURL, "#") {
				continue
			}
			u, err := url.Parse(rawURL)
			if err != nil {
				continue
			}
			resolved := baseURL.ResolveReference(u).String()
			links = append(links, resolved)
		}
	}
	return links
}

// ScrapeLinksDynamic extracts all links matching the CSS selector from the homepage using ChromeDP.
func ScrapeLinksDynamic(pageURL string, selector string) ([]string, error) {
	if selector == "" {
		selector = "a"
	}
	
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox,
	)
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 35*time.Second)
	defer cancel()

	var links []string
	err := chromedp.Run(ctx,
		chromedp.Navigate(pageURL),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll("` + selector + `"))
				.map(a => a.href)
				.filter(href => href && !href.startsWith('javascript:') && !href.startsWith('#'))
		`, &links),
	)
	if err != nil {
		return nil, fmt.Errorf("dynamic link scraping failed: %w", err)
	}
	return links, nil
}

// FilterVideoLinks filters and de-duplicates candidate URLs.
func FilterVideoLinks(links []string, baseURL *url.URL, regexFilter string) []string {
	var filtered []string
	seen := make(map[string]bool)
	
	var filterRe *regexp.Regexp
	if regexFilter != "" {
		var err error
		filterRe, err = regexp.Compile(regexFilter)
		if err != nil {
			fmt.Printf("Warning: invalid filter regex: %v. Ignoring filter.\n", err)
			filterRe = nil
		}
	}
	
	ignoredSuffixes := []string{
		".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".ico",
		".css", ".js", ".json", ".xml",
		".pdf", ".zip", ".tar.gz", ".rar", ".7z",
		".mp4", ".mkv", ".avi", ".mov", ".mp3", ".wav",
	}

	for _, link := range links {
		u, err := url.Parse(link)
		if err != nil {
			continue
		}
		
		if u.Host != baseURL.Host {
			continue
		}
		
		u.Fragment = ""
		normalized := u.String()
		
		if seen[normalized] {
			continue
		}
		
		lowerPath := strings.ToLower(u.Path)
		isAsset := false
		for _, suffix := range ignoredSuffixes {
			if strings.HasSuffix(lowerPath, suffix) {
				isAsset = true
				break
			}
		}
		if isAsset {
			continue
		}
		
		if filterRe != nil && !filterRe.MatchString(normalized) {
			continue
		}
		
		seen[normalized] = true
		filtered = append(filtered, normalized)
	}
	
	return filtered
}

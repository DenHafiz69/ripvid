package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func main() {
	dynamicFlag := flag.Bool("dynamic", false, "Force using dynamic scraper (headless browser)")
	flag.BoolVar(dynamicFlag, "d", false, "Force using dynamic scraper (headless browser)")

	outputDirFlag := flag.String("output", "", "Directory to save downloaded videos (default: ~/Videos/ripvid)")
	flag.StringVar(outputDirFlag, "o", "", "Directory to save downloaded videos (default: ~/Videos/ripvid)")

	homeFlag := flag.Bool("home", false, "Scour homepage for possible videos, open each, and download them")
	flag.BoolVar(homeFlag, "h", false, "Scour homepage for possible videos, open each, and download them")

	filterFlag := flag.String("filter", "", "Filter candidate URLs using regex (homepage mode only)")
	flag.StringVar(filterFlag, "f", "", "Filter candidate URLs using regex (homepage mode only)")

	selectorFlag := flag.String("selector", "", "CSS selector to query anchor tags containing video URLs (homepage mode only)")
	flag.StringVar(selectorFlag, "s", "", "CSS selector to query anchor tags containing video URLs (homepage mode only)")

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: ripvid [options] <URL>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	pageURL := args[0]

	// Resolve output directory
	outputDir := *outputDirFlag
	if outputDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			outputDir = filepath.Join("Videos", "ripvid")
		} else {
			outputDir = filepath.Join(homeDir, "Videos", "ripvid")
		}
	} else if strings.HasPrefix(outputDir, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			outputDir = filepath.Join(homeDir, outputDir[2:])
		}
	}

	if *homeFlag {
		parsedURL, err := url.Parse(pageURL)
		if err != nil {
			fmt.Printf("Error parsing homepage URL: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Scouring homepage for video links: %s...\n", pageURL)
		rawLinks, err := ScrapeLinks(pageURL, *selectorFlag)
		if err != nil {
			fmt.Printf("Error scouring homepage links: %v\n", err)
			os.Exit(1)
		}

		videoLinks := FilterVideoLinks(rawLinks, parsedURL, *filterFlag)
		if len(videoLinks) == 0 {
			fmt.Println("No candidate video URLs found on the homepage.")
			os.Exit(0)
		}

		fmt.Printf("Found %d video page candidate(s):\n", len(videoLinks))
		for i, link := range videoLinks {
			fmt.Printf("  [%d] %s\n", i+1, link)
		}

		fmt.Print("\nEnter selection (e.g. '1', '1-5', '1,3,5-7', or 'all' / empty): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')

		selectedIndices, err := ParseSelection(input, len(videoLinks))
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		if len(selectedIndices) == 0 {
			fmt.Println("No videos selected for download.")
			os.Exit(0)
		}

		fmt.Printf("\nSelected %d video(s) for download.\n", len(selectedIndices))

		// Download loop
		for rank, idx := range selectedIndices {
			link := videoLinks[idx]
			fmt.Printf("\n--- [%d/%d] Processing video page: %s ---\n", rank+1, len(selectedIndices), link)
			
			var result *ScrapeResult
			var err error
			
			if *dynamicFlag {
				result, err = ScrapePageDynamic(link)
			} else {
				result, err = ScrapePage(link)
				if err == nil && len(result.M3U8s) == 0 {
					fmt.Println("No streams found in static HTML. Falling back to dynamic browser rendering...")
					result, err = ScrapePageDynamic(link)
				}
			}
			
			if err != nil {
				fmt.Printf("Error scraping page %s: %v\n", link, err)
				continue
			}

			if len(result.M3U8s) == 0 {
				fmt.Printf("No stream (.m3u8) found on page %s\n", link)
				continue
			}

			title := result.Title
			if title == "" {
				title = fmt.Sprintf("video_%s_%d", time.Now().Format("20060102_150405"), idx+1)
			}
			fmt.Printf("Found title: %q\n", title)
			fmt.Printf("Found %d stream(s), downloading first stream...\n", len(result.M3U8s))
			
			err = DownloadVideo(result.M3U8s[0], title, outputDir)
			if err != nil {
				fmt.Printf("Error downloading video %s: %v\n", title, err)
			} else {
				fmt.Printf("Successfully downloaded %s!\n", title)
			}
		}
		fmt.Println("\nFinished homepage download tasks!")
		os.Exit(0)
	}

	var result *ScrapeResult
	var err error

	if *dynamicFlag {
		fmt.Printf("Forcing dynamic scraping for: %s...\n", pageURL)
		result, err = ScrapePageDynamic(pageURL)
	} else {
		fmt.Printf("Fetching page (static): %s...\n", pageURL)
		result, err = ScrapePage(pageURL)
		if err == nil && len(result.M3U8s) == 0 {
			fmt.Println("No streams found in static HTML. Falling back to dynamic browser rendering...")
			result, err = ScrapePageDynamic(pageURL)
		}
	}

	if err != nil {
		fmt.Printf("Error scraping page: %v\n", err)
		os.Exit(1)
	}

	title := result.Title
	if title == "" {
		title = fmt.Sprintf("video_%s", time.Now().Format("20060102_150405"))
		fmt.Printf("No title found, falling back to: %s\n", title)
	} else {
		fmt.Printf("Found title: %q\n", title)
	}

	if len(result.M3U8s) == 0 {
		fmt.Println("Error: No .m3u8 files found on the page.")
		os.Exit(1)
	}

	fmt.Printf("Found %d stream(s):\n", len(result.M3U8s))
	for i, u := range result.M3U8s {
		fmt.Printf("  [%d] %s\n", i+1, u)
	}

	// Download the first one
	targetM3U8 := result.M3U8s[0]
	fmt.Printf("\nStarting download for stream [1]...\n")
	err = DownloadVideo(targetM3U8, title, outputDir)
	if err != nil {
		fmt.Printf("Error downloading video: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Download completed successfully!")
}

// ParseSelection parses selection input (e.g., "1,3,5-7", "all") and returns 0-indexed selected indices.
func ParseSelection(input string, max int) ([]int, error) {
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" || input == "all" {
		var all []int
		for i := 0; i < max; i++ {
			all = append(all, i)
		}
		return all, nil
	}

	seen := make(map[int]bool)
	var result []int

	parts := strings.Split(input, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range format: %s", part)
			}
			startStr := strings.TrimSpace(rangeParts[0])
			endStr := strings.TrimSpace(rangeParts[1])

			var start, end int
			_, err1 := fmt.Sscan(startStr, &start)
			_, err2 := fmt.Sscan(endStr, &end)
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("invalid numbers in range: %s", part)
			}

			if start > end {
				start, end = end, start
			}

			for i := start; i <= end; i++ {
				idx := i - 1
				if idx >= 0 && idx < max {
					if !seen[idx] {
						seen[idx] = true
						result = append(result, idx)
					}
				} else {
					return nil, fmt.Errorf("index out of bounds (1-%d): %d", max, i)
				}
			}
		} else {
			var val int
			_, err := fmt.Sscan(part, &val)
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", part)
			}
			idx := val - 1
			if idx >= 0 && idx < max {
				if !seen[idx] {
					seen[idx] = true
					result = append(result, idx)
				}
			} else {
				return nil, fmt.Errorf("index out of bounds (1-%d): %d", max, val)
			}
		}
	}

	sort.Ints(result)
	return result, nil
}

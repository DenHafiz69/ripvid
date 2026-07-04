package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	dynamicFlag := flag.Bool("dynamic", false, "Force using dynamic scraper (headless browser)")
	flag.BoolVar(dynamicFlag, "d", false, "Force using dynamic scraper (headless browser)")

	outputDirFlag := flag.String("output", "", "Directory to save downloaded videos (default: ~/Videos/ripvid)")
	flag.StringVar(outputDirFlag, "o", "", "Directory to save downloaded videos (default: ~/Videos/ripvid)")

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

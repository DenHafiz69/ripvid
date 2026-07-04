# ripvid

A Go-based command-line tool designed to extract `.m3u8` video stream URLs from webpages and download them using `yt-dlp`. It is specifically built for websites that are not natively supported by `yt-dlp` extractors.

## Features

- **Dual-Mode Scraper:**
  - **Static Scraper:** Fetches the webpage's HTML directly and performs high-speed regex-based extraction of absolute/relative `.m3u8` URLs and metadata titles.
  - **Dynamic Scraper (Chromedp Fallback):** Automatically launches a headless Chrome/Chromium instance to execute JavaScript, monitor network traffic, and capture dynamic/deferred `.m3u8` requests.
- **Homepage Mode:**
  - Scour index/homepages to find candidate video detail links using the `--home` / `-h` flag.
  - Customize link extraction target using CSS selectors with `--selector` / `-s`.
  - Filter candidate links using regex patterns via `--filter` / `-f`.
- **Interactive Selection Prompt:**
  - Prompt user to pick which crawled videos to download. Supports ranges (e.g. `1-5`), comma-separated numbers (e.g. `1,3,5-7`), individual selections, or `all`.
- **Smart Title Extraction:** Automatically fetches titles from standard `<title>` tags, `og:title` metadata, or `twitter:title` metadata to name the output file.
- **Safe Filename Sanitization:** Sanitizes titles to remove characters that are illegal or problematic in OS filesystems.
- **Subprocess Integration:** Runs `yt-dlp` inside a subprocess, streaming download progress (speed, size, eta) directly to your terminal.

## Prerequisites

1. **Go** (to compile the project)
2. [**yt-dlp**](https://github.com/yt-dlp/yt-dlp) installed and available in your system path
3. **Chrome or Chromium** installed (required for the dynamic scraper fallback and dynamic link extraction)

## Installation

Clone the repository and build the binary:

```bash
go build -o ripvid
```

## Usage

### Basic Usage (Auto-Fallback)

Simply run the tool with the page URL. It will first attempt a static crawl. If no streams are found, it automatically boots the headless browser to capture dynamic traffic:

```bash
./ripvid https://example.com/video-page
```

### Force Dynamic Scraping

If you know a website requires rendering JavaScript to even generate page titles or start the player, you can force the dynamic scraper using the `-d` or `--dynamic` flag:

```bash
./ripvid -d https://example.com/dynamic-video-page
```

### Specifying Output Directory

By default, downloads are saved to `~/Videos/ripvid`. You can specify a custom output directory using the `-o` or `--output` flag:

```bash
./ripvid -o /path/to/my/downloads https://example.com/video-page
```

### Homepage Mode

To scour a homepage for candidate video pages and download selected items:

```bash
./ripvid --home https://example.com/videos
```

#### Filtering Candidate URLs
You can filter found candidate video URLs using a regular expression with `-f` or `--filter`:

```bash
./ripvid -h -f "watch\?v=" https://example.com/videos
```

#### Custom CSS Selector
To extract video URLs from specific elements on the page (for instance, links inside a specific card container):

```bash
./ripvid -h -s ".video-grid a" https://example.com/videos
```

Once candidate URLs are fetched, you will be prompted to select which videos to download:
```
Found 10 video page candidate(s):
  [1] https://example.com/videos/page1
  [2] https://example.com/videos/page2
  ...

Enter selection (e.g. '1', '1-5', '1,3,5-7', or 'all' / empty): 1,3-5
```

## How It Works

1. **Title Extraction:** The webpage's document or OpenGraph title is retrieved and sanitized (e.g., `Video: How to Code?` becomes `Video_ How to Code.mp4`).
2. **Stream Locating:** 
   - Static search parses page HTML for URLs matching `.m3u8` (unescaping JavaScript JSON payloads).
   - Dynamic search mounts Chrome DevTools Protocol network listeners for request interception.
3. **Download:** The first extracted `.m3u8` link is dispatched to `yt-dlp`, remuxing the stream segments into a final `.mp4` video.

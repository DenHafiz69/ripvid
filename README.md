# ripvid

A Go-based command-line tool designed to extract `.m3u8` video stream URLs from webpages and download them using `yt-dlp`. It is specifically built for websites that are not natively supported by `yt-dlp` extractors.

## Features

- **Dual-Mode Scraper:**
  - **Static Scraper:** Fetches the webpage's HTML directly and performs high-speed regex-based extraction of absolute/relative `.m3u8` URLs and metadata titles.
  - **Dynamic Scraper (Chromedp Fallback):** Automatically launches a headless Chrome/Chromium instance to execute JavaScript, monitor network traffic, and capture dynamic/deferred `.m3u8` requests.
- **Smart Title Extraction:** Automatically fetches titles from standard `<title>` tags, `og:title` metadata, or `twitter:title` metadata to name the output file.
- **Safe Filename Sanitization:** Sanitizes titles to remove characters that are illegal or problematic in OS filesystems.
- **Subprocess Integration:** Runs `yt-dlp` inside a subprocess, streaming download progress (speed, size, eta) directly to your terminal.

## Prerequisites

1. **Go** (to compile the project)
2. [**yt-dlp**](https://github.com/yt-dlp/yt-dlp) installed and available in your system path
3. **Chrome or Chromium** installed (required for the dynamic scraper fallback)

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

## How It Works

1. **Title Extraction:** The webpage's document or OpenGraph title is retrieved and sanitized (e.g., `Video: How to Code?` becomes `Video_ How to Code.mp4`).
2. **Stream Locating:** 
   - Static search parses page HTML for URLs matching `.m3u8` (unescaping JavaScript JSON payloads).
   - Dynamic search mounts Chrome DevTools Protocol network listeners for request interception.
3. **Download:** The first extracted `.m3u8` link is dispatched to `yt-dlp`, remuxing the stream segments into a final `.mp4` video.

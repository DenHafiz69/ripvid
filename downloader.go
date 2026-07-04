package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// SanitizeFilename removes characters that are illegal or problematic in filenames.
func SanitizeFilename(title string) string {
	if title == "" {
		return "video"
	}

	// Remove control characters and illegal filename characters: / \ : * ? " < > |
	reg := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	safe := reg.ReplaceAllString(title, "_")

	// Replace multiple consecutive underscores with a single one
	regMulti := regexp.MustCompile(`_+`)
	safe = regMulti.ReplaceAllString(safe, "_")

	// Trim whitespace and underscores
	safe = strings.Trim(safe, " _")

	// Truncate if too long
	if len(safe) > 150 {
		safe = safe[:150]
	}

	if safe == "" {
		return "video"
	}
	return safe
}

// DownloadVideo downloads the stream using yt-dlp
func DownloadVideo(m3u8URL, title string) error {
	safeTitle := SanitizeFilename(title)
	outputFile := safeTitle + ".mp4"

	fmt.Printf("Downloading stream: %s\n", m3u8URL)
	fmt.Printf("Saving as: %s\n", outputFile)

	// Build cmd: yt-dlp -o "Title.mp4" "m3u8_url"
	cmd := exec.Command("yt-dlp", "-o", outputFile, m3u8URL)

	// Stream stdout and stderr directly to our CLI's stdout/stderr so the user can see progress
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("yt-dlp download failed: %w", err)
	}

	return nil
}

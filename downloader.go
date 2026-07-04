package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// DownloadVideo downloads the stream using yt-dlp to the specified output directory.
func DownloadVideo(m3u8URL, title, outputDir string) error {
	safeTitle := SanitizeFilename(title)
	
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, safeTitle+".mp4")

	fmt.Printf("Downloading stream: %s\n", m3u8URL)
	fmt.Printf("Saving as: %s\n", outputPath)

	// Build cmd: yt-dlp -o "path/to/title.mp4" "m3u8_url"
	cmd := exec.Command("yt-dlp", "-o", outputPath, m3u8URL)

	// Stream stdout and stderr directly to our CLI's stdout/stderr so the user can see progress
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("yt-dlp download failed: %w", err)
	}

	return nil
}

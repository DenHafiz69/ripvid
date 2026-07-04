package main

import (
	"net/url"
	"reflect"
	"testing"
)

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "Standard Title",
			html: `<html><head><title>My Cool Video</title></head></html>`,
			want: "My Cool Video",
		},
		{
			name: "Title with attributes",
			html: `<html><head><title class="header">My Cool Video 2</title></head></html>`,
			want: "My Cool Video 2",
		},
		{
			name: "og:title first",
			html: `<html><head>
				<meta property="og:title" content="OG Cool Video" />
				<title>Document Title</title>
			</head></html>`,
			want: "OG Cool Video",
		},
		{
			name: "twitter:title fallback",
			html: `<html><head>
				<meta name="twitter:title" content="Twitter Cool Video" />
			</head></html>`,
			want: "Twitter Cool Video",
		},
		{
			name: "HTML Entities in Title",
			html: `<html><head><title>Cool &amp; Amazing &quot;Video&quot;</title></head></html>`,
			want: `Cool & Amazing "Video"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitle(tt.html)
			if got != tt.want {
				t.Errorf("extractTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractM3U8s(t *testing.T) {
	baseURL, _ := url.Parse("https://example.com/page.html")
	tests := []struct {
		name string
		html string
		want []string
	}{
		{
			name: "Absolute URL in plain text",
			html: `Check out this stream: https://video.server.com/live/stream.m3u8`,
			want: []string{"https://video.server.com/live/stream.m3u8"},
		},
		{
			name: "Relative URL resolving",
			html: `var url = "/media/playlist.m3u8";`,
			want: []string{"https://example.com/media/playlist.m3u8"},
		},
		{
			name: "Escaped slashes inside JSON",
			html: `{"streamUrl": "https:\/\/video.server.com\/live\/stream.m3u8?token=abc"}`,
			want: []string{"https://video.server.com/live/stream.m3u8?token=abc"},
		},
		{
			name: "HTML entity url encoding",
			html: `<a href="https://example.com/stream.m3u8?key=val&amp;auth=xyz">stream</a>`,
			want: []string{"https://example.com/stream.m3u8?key=val&auth=xyz"},
		},
		{
			name: "Multiple mixed URLs and deduplication",
			html: `
				"https://example.com/a.m3u8"
				"https://example.com/a.m3u8"
				"https://example.com/b.m3u8"
				"/c.m3u8"
			`,
			want: []string{
				"https://example.com/a.m3u8",
				"https://example.com/b.m3u8",
				"https://example.com/c.m3u8",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractM3U8s(tt.html, baseURL)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractM3U8s() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{
			name:  "Clean title",
			title: "Simple Video Title",
			want:  "Simple Video Title",
		},
		{
			name:  "Title with bad characters",
			title: `Video: How to / Code * Test?`,
			want:  "Video_ How to _ Code _ Test",
		},
		{
			name:  "Title with multiple spaces and underscores",
			title: "Video___Test   Title",
			want:  "Video_Test   Title",
		},
		{
			name:  "Empty title",
			title: "",
			want:  "video",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeFilename(tt.title)
			if got != tt.want {
				t.Errorf("SanitizeFilename() = %q, want %q", got, tt.want)
			}
		})
	}
}

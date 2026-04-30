package site

import (
	"strings"
	"testing"

	"github.com/nevenkitasuno/blog-ssg/internal/domain"
)

func TestEntryLabel(t *testing.T) {
	tests := []struct {
		name  string
		entry domain.Entry
		want  string
	}{
		{
			name:  "month only",
			entry: domain.Entry{Month: 4, Title: "Post"},
			want:  "04 Post",
		},
		{
			name:  "month and day",
			entry: domain.Entry{Month: 4, Day: 9, Title: "Post"},
			want:  "04 09 Post",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := entryLabel(tt.entry); got != tt.want {
				t.Fatalf("entryLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderTopicPreviewHTML(t *testing.T) {
	got := string(renderTopicPreviewHTML("Preview with **bold** text"))
	want := "<p>Preview with <strong>bold</strong> text</p>\n"
	if got != want {
		t.Fatalf("renderTopicPreviewHTML() = %q, want %q", got, want)
	}
}

func TestResolveTopicLink(t *testing.T) {
	tests := []struct {
		name string
		link domain.TopicLink
		want string
	}{
		{
			name: "external",
			link: domain.TopicLink{Label: "Repo", Target: "https://example.com", External: true},
			want: "https://example.com",
		},
		{
			name: "entry first page",
			link: domain.TopicLink{Label: "Entry", Target: "2025 12 11 Entry with date/1.md"},
			want: "2025-12-11-entry-with-date/index.html",
		},
		{
			name: "entry later page",
			link: domain.TopicLink{Label: "Page", Target: "2026 03 Коты/2.md"},
			want: "2026-03-коты/2/index.html",
		},
		{
			name: "topic meta page",
			link: domain.TopicLink{Label: "Meta", Target: "meta/Ktotam.md"},
			want: "meta/ktotam/index.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveTopicLink(tt.link); got != tt.want {
				t.Fatalf("resolveTopicLink() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTopicBannerURL(t *testing.T) {
	topic := domain.Topic{
		Assets: []domain.Asset{
			{Name: "banner.jpg"},
			{Name: "top_banner.jpg"},
		},
	}

	got := topicBannerURL(topic, func(name string) string {
		return "meta/" + name
	})
	want := "meta/top_banner.jpg"
	if got != want {
		t.Fatalf("topicBannerURL() = %q, want %q", got, want)
	}
}

func TestRenderTopicThemeCSS(t *testing.T) {
	theme := domain.TopicTheme{
		Background: "#f5f7fa",
		Accent:     "#123456",
		Heading:    "#654321",
		Muted:      "slategray",
		Surface:    "rgba(1, 2, 3, 0.4)",
		Border:     "rgba(10, 20, 30, 0.5)",
		CodeBG:     "#eeeeee",
		CodeBorder: "#cccccc",
	}

	got := string(renderTopicThemeCSS(theme))
	for _, want := range []string{
		"--color-background: #f5f7fa;",
		"--color-accent: #123456;",
		"--color-heading: #654321;",
		"--color-muted: slategray;",
		"--color-surface: rgba(1, 2, 3, 0.4);",
		"--color-border: rgba(10, 20, 30, 0.5);",
		"--color-code-bg: #eeeeee;",
		"--color-code-border: #cccccc;",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("renderTopicThemeCSS() = %q, missing %q", got, want)
		}
	}
}

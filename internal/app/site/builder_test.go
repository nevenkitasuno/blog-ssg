package site

import (
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

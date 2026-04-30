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

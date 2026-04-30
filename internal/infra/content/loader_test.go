package content

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoaderSupportsOptionalDayInEntryDirectory(t *testing.T) {
	root := t.TempDir()
	topicDir := filepath.Join(root, "Tech")
	if err := os.MkdirAll(topicDir, 0o755); err != nil {
		t.Fatalf("mkdir topic: %v", err)
	}

	writeEntryPage(t, filepath.Join(topicDir, "2026 04 Post without day", "1.md"), "First preview")
	writeEntryPage(t, filepath.Join(topicDir, "2026 04 09 Post with day", "1.md"), "Second preview")
	writeEntryPage(t, filepath.Join(topicDir, "2026 04 21 Later dated post", "1.md"), "Third preview")

	blog, err := NewLoader(root).Load()
	if err != nil {
		t.Fatalf("load blog: %v", err)
	}

	if len(blog.Topics) != 1 {
		t.Fatalf("topics = %d, want 1", len(blog.Topics))
	}

	entries := blog.Topics[0].Entries
	if len(entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(entries))
	}

	if entries[0].Day != 21 || entries[0].Title != "Later dated post" {
		t.Fatalf("first entry = %+v, want day 21", entries[0])
	}
	if entries[1].Day != 9 || entries[1].Title != "Post with day" {
		t.Fatalf("second entry = %+v, want day 9", entries[1])
	}
	if entries[2].Day != 0 || entries[2].Title != "Post without day" {
		t.Fatalf("third entry = %+v, want no day", entries[2])
	}
}

func TestLoaderExtractsPreviewFromFirstParagraph(t *testing.T) {
	root := t.TempDir()
	topicDir := filepath.Join(root, "Gallery")
	if err := os.MkdirAll(topicDir, 0o755); err != nil {
		t.Fatalf("mkdir topic: %v", err)
	}

	writeEntryPage(t, filepath.Join(topicDir, "2025 12 11 Entry with date", "1.md"), `---
tags:
  - photos
---

# Heading

First paragraph for preview
continues here.

- list item

Second paragraph`)

	blog, err := NewLoader(root).Load()
	if err != nil {
		t.Fatalf("load blog: %v", err)
	}

	if len(blog.Topics) != 1 || len(blog.Topics[0].Entries) != 1 {
		t.Fatalf("unexpected blog shape: %+v", blog)
	}

	got := blog.Topics[0].Entries[0].Preview
	want := "First paragraph for preview continues here."
	if got != want {
		t.Fatalf("preview = %q, want %q", got, want)
	}
}

func TestLoaderRejectsInvalidDayInEntryDirectory(t *testing.T) {
	root := t.TempDir()
	topicDir := filepath.Join(root, "Tech")
	if err := os.MkdirAll(topicDir, 0o755); err != nil {
		t.Fatalf("mkdir topic: %v", err)
	}

	writeEntryPage(t, filepath.Join(topicDir, "2026 04 99 Invalid day", "1.md"), "# invalid")

	blog, err := NewLoader(root).Load()
	if err != nil {
		t.Fatalf("load blog: %v", err)
	}

	if len(blog.Topics) != 0 {
		t.Fatalf("topics = %d, want 0", len(blog.Topics))
	}
}

func writeEntryPage(t *testing.T, path, body string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir entry: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write page: %v", err)
	}
}

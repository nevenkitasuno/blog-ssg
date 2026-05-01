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

func TestLoaderExtractsTopicLinks(t *testing.T) {
	root := t.TempDir()
	topicDir := filepath.Join(root, "Gallery")
	if err := os.MkdirAll(filepath.Join(topicDir, "meta"), 0o755); err != nil {
		t.Fatalf("mkdir meta: %v", err)
	}

	writeEntryPage(t, filepath.Join(topicDir, "2025 12 11 Entry with date", "1.md"), "Preview")
	if err := os.WriteFile(filepath.Join(topicDir, "meta", "Links.md"), []byte(`- [[2025 12 11 Entry with date/1.md|Internal]]
- [External](https://example.com)`), 0o644); err != nil {
		t.Fatalf("write links: %v", err)
	}

	blog, err := NewLoader(root).Load()
	if err != nil {
		t.Fatalf("load blog: %v", err)
	}

	links := blog.Topics[0].Links
	if len(links) != 2 {
		t.Fatalf("links = %d, want 2", len(links))
	}
	if links[0].Label != "Internal" || links[0].Target != "2025 12 11 Entry with date/1.md" || links[0].External {
		t.Fatalf("first link = %+v", links[0])
	}
	if links[1].Label != "External" || links[1].Target != "https://example.com" || !links[1].External {
		t.Fatalf("second link = %+v", links[1])
	}
}

func TestLoaderLoadsTopicMetaPagesAndAssets(t *testing.T) {
	root := t.TempDir()
	topicDir := filepath.Join(root, "Gallery")
	if err := os.MkdirAll(filepath.Join(topicDir, "meta"), 0o755); err != nil {
		t.Fatalf("mkdir meta: %v", err)
	}

	writeEntryPage(t, filepath.Join(topicDir, "2025 12 11 Entry with date", "1.md"), "Preview")
	writeEntryPage(t, filepath.Join(topicDir, "meta", "About.md"), "About topic")
	if err := os.WriteFile(filepath.Join(topicDir, "meta", "banner.jpg"), []byte("jpg"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topicDir, "meta", "Links.md"), []byte("- [External](https://example.com)"), 0o644); err != nil {
		t.Fatalf("write links: %v", err)
	}

	blog, err := NewLoader(root).Load()
	if err != nil {
		t.Fatalf("load blog: %v", err)
	}

	topic := blog.Topics[0]
	if len(topic.Meta) != 1 {
		t.Fatalf("meta pages = %d, want 1", len(topic.Meta))
	}
	if topic.Meta[0].Name != "About.md" || topic.Meta[0].Slug != "about" || topic.Meta[0].Title != "About" {
		t.Fatalf("meta page = %+v", topic.Meta[0])
	}
	if len(topic.Assets) != 1 || topic.Assets[0].Name != "banner.jpg" {
		t.Fatalf("meta assets = %+v", topic.Assets)
	}
}

func TestLoaderLoadsTopicTheme(t *testing.T) {
	root := t.TempDir()
	topicDir := filepath.Join(root, "Gallery")
	if err := os.MkdirAll(filepath.Join(topicDir, "meta"), 0o755); err != nil {
		t.Fatalf("mkdir meta: %v", err)
	}

	writeEntryPage(t, filepath.Join(topicDir, "2025 12 11 Entry with date", "1.md"), "Preview")
	if err := os.WriteFile(filepath.Join(topicDir, "meta", "Config.yaml"), []byte(`background: "#f5f7fa"
text: "#111111"
accent: "#123456"
heading: "#654321"
muted: slategray
surface: "rgba(1, 2, 3, 0.4)"
border: "rgba(10, 20, 30, 0.5)"
code_bg: "#eeeeee"
code_border: "#cccccc"`), 0o644); err != nil {
		t.Fatalf("write theme: %v", err)
	}

	blog, err := NewLoader(root).Load()
	if err != nil {
		t.Fatalf("load blog: %v", err)
	}

	theme := blog.Topics[0].Theme
	if theme.Background != "#f5f7fa" || theme.Text != "#111111" || theme.Accent != "#123456" || theme.Heading != "#654321" || theme.Muted != "slategray" {
		t.Fatalf("theme = %+v", theme)
	}
	if theme.Surface != "rgba(1, 2, 3, 0.4)" || theme.Border != "rgba(10, 20, 30, 0.5)" {
		t.Fatalf("theme = %+v", theme)
	}
	if theme.CodeBG != "#eeeeee" || theme.CodeBorder != "#cccccc" {
		t.Fatalf("theme = %+v", theme)
	}
}

func TestLoaderUsesConfiguredTopicLinkName(t *testing.T) {
	root := t.TempDir()
	topicDir := filepath.Join(root, "Gallery")
	if err := os.MkdirAll(filepath.Join(topicDir, "meta"), 0o755); err != nil {
		t.Fatalf("mkdir meta: %v", err)
	}

	writeEntryPage(t, filepath.Join(topicDir, "2025 12 11 Entry with date", "1.md"), "Preview")
	if err := os.WriteFile(filepath.Join(topicDir, "meta", "Config.yaml"), []byte("link_name: configured-link-name\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	blog, err := NewLoader(root).Load()
	if err != nil {
		t.Fatalf("load blog: %v", err)
	}

	if got := blog.Topics[0].Slug; got != "configured-link-name" {
		t.Fatalf("topic slug = %q, want %q", got, "configured-link-name")
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

package content

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"gopkg.in/yaml.v2"

	"github.com/nevenkitasuno/blog-ssg/internal/domain"
)

var entryPattern = regexp.MustCompile(`^(\d{4})\s+(\d{2})(?:\s+(\d{2}))?\s+(.+)$`)

type Loader struct {
	contentDir string
}

type frontMatter struct {
	Tags []string `yaml:"tags"`
}

func NewLoader(contentDir string) *Loader {
	return &Loader{contentDir: contentDir}
}

func (l *Loader) Load() (domain.Blog, error) {
	entries, err := os.ReadDir(l.contentDir)
	if err != nil {
		return domain.Blog{}, fmt.Errorf("read content directory: %w", err)
	}

	blog := domain.Blog{Topics: make([]domain.Topic, 0, len(entries))}
	for _, topicDir := range entries {
		if !topicDir.IsDir() {
			continue
		}

		topic, ok, err := l.loadTopic(topicDir.Name())
		if err != nil {
			return domain.Blog{}, err
		}
		if ok {
			blog.Topics = append(blog.Topics, topic)
		}
	}

	slices.SortFunc(blog.Topics, func(left, right domain.Topic) int {
		return strings.Compare(left.Name, right.Name)
	})

	return blog, nil
}

func (l *Loader) loadTopic(name string) (domain.Topic, bool, error) {
	entries, err := os.ReadDir(filepath.Join(l.contentDir, name))
	if err != nil {
		return domain.Topic{}, false, fmt.Errorf("read topic %q: %w", name, err)
	}

	topic := domain.Topic{
		Name:    name,
		Slug:    slugify(name),
		Entries: make([]domain.Entry, 0, len(entries)),
	}

	for _, entryDir := range entries {
		if !entryDir.IsDir() {
			continue
		}

		entry, ok, err := l.loadEntry(name, entryDir.Name())
		if err != nil {
			return domain.Topic{}, false, err
		}
		if ok {
			topic.Entries = append(topic.Entries, entry)
		}
	}

	if len(topic.Entries) == 0 {
		return domain.Topic{}, false, nil
	}

	slices.SortFunc(topic.Entries, func(left, right domain.Entry) int {
		if left.Year != right.Year {
			return right.Year - left.Year
		}
		if left.Month != right.Month {
			return right.Month - left.Month
		}
		if left.Day != right.Day {
			return right.Day - left.Day
		}
		return strings.Compare(left.Title, right.Title)
	})

	return topic, true, nil
}

func (l *Loader) loadEntry(topicName, dirName string) (domain.Entry, bool, error) {
	matches := entryPattern.FindStringSubmatch(dirName)
	if matches == nil {
		return domain.Entry{}, false, nil
	}

	year, err := strconv.Atoi(matches[1])
	if err != nil {
		return domain.Entry{}, false, fmt.Errorf("parse year in %q: %w", dirName, err)
	}

	month, err := strconv.Atoi(matches[2])
	if err != nil {
		return domain.Entry{}, false, fmt.Errorf("parse month in %q: %w", dirName, err)
	}
	if month < 1 || month > 12 {
		return domain.Entry{}, false, nil
	}

	day := 0
	if matches[3] != "" {
		day, err = strconv.Atoi(matches[3])
		if err != nil {
			return domain.Entry{}, false, fmt.Errorf("parse day in %q: %w", dirName, err)
		}
		if day < 1 || day > 31 {
			return domain.Entry{}, false, nil
		}
	}

	title := strings.TrimSpace(matches[4])
	if title == "" {
		return domain.Entry{}, false, nil
	}

	files, err := os.ReadDir(filepath.Join(l.contentDir, topicName, dirName))
	if err != nil {
		return domain.Entry{}, false, fmt.Errorf("read entry %q: %w", dirName, err)
	}

	entry := domain.Entry{
		Name:   dirName,
		Slug:   slugify(dirName),
		Year:   year,
		Month:  month,
		Day:    day,
		Title:  title,
		Tags:   nil,
		Assets: make([]domain.Asset, 0),
		Pages:  make([]domain.Page, 0, len(files)),
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if filepath.Ext(file.Name()) != ".md" {
			entry.Assets = append(entry.Assets, domain.Asset{
				Name: file.Name(),
				Path: filepath.Join(l.contentDir, topicName, dirName, file.Name()),
			})
			continue
		}

		page, ok, err := l.loadPage(topicName, dirName, file.Name())
		if err != nil {
			return domain.Entry{}, false, err
		}
		if ok {
			entry.Pages = append(entry.Pages, page)
		}
	}

	if len(entry.Pages) == 0 {
		return domain.Entry{}, false, nil
	}

	slices.SortFunc(entry.Pages, func(left, right domain.Page) int {
		return left.Number - right.Number
	})
	slices.SortFunc(entry.Assets, func(left, right domain.Asset) int {
		return strings.Compare(left.Name, right.Name)
	})
	if len(entry.Pages) > 0 && entry.Pages[0].Number == 1 {
		tags, err := extractTagsFromFile(entry.Pages[0].File.Path)
		if err != nil {
			return domain.Entry{}, false, fmt.Errorf("parse tags in %q: %w", entry.Pages[0].File.Path, err)
		}
		entry.Tags = tags
	}

	return entry, true, nil
}

func (l *Loader) loadPage(topicName, entryName, fileName string) (domain.Page, bool, error) {
	if filepath.Ext(fileName) != ".md" {
		return domain.Page{}, false, nil
	}

	number, err := strconv.Atoi(strings.TrimSuffix(fileName, filepath.Ext(fileName)))
	if err != nil || number < 1 {
		return domain.Page{}, false, nil
	}

	path := filepath.Join(l.contentDir, topicName, entryName, fileName)
	body, err := os.ReadFile(path)
	if err != nil {
		return domain.Page{}, false, fmt.Errorf("read markdown file %q: %w", path, err)
	}

	markdownBody, _, err := splitFrontMatter(string(body))
	if err != nil {
		return domain.Page{}, false, fmt.Errorf("parse front matter in %q: %w", path, err)
	}

	trimmed := strings.TrimSpace(markdownBody)
	if trimmed == "" {
		return domain.Page{}, false, nil
	}

	return domain.Page{
		Number: number,
		File: domain.MarkdownFile{
			Path: path,
			Body: trimmed,
		},
	}, true, nil
}

func extractTags(body string) ([]string, error) {
	_, rawFrontMatter, err := splitFrontMatter(body)
	if err != nil || rawFrontMatter == "" {
		return nil, err
	}

	var meta frontMatter
	if err := yaml.Unmarshal([]byte(rawFrontMatter), &meta); err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	tags := make([]string, 0, len(meta.Tags))
	for _, tag := range meta.Tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		tags = append(tags, trimmed)
	}

	slices.SortFunc(tags, func(left, right string) int {
		return strings.Compare(strings.ToLower(left), strings.ToLower(right))
	})

	return tags, nil
}

func extractTagsFromFile(path string) ([]string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return extractTags(string(body))
}

func splitFrontMatter(body string) (markdownBody string, rawFrontMatter string, err error) {
	normalized := strings.ReplaceAll(body, "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---\n") {
		return normalized, "", nil
	}

	rest := normalized[len("---\n"):]
	end := strings.Index(rest, "\n---\n")
	if end == -1 {
		return "", "", fmt.Errorf("unterminated yaml front matter")
	}

	rawFrontMatter = rest[:end]
	markdownBody = rest[end+len("\n---\n"):]
	return markdownBody, rawFrontMatter, nil
}

func slugify(value string) string {
	var builder strings.Builder
	lastDash := false

	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastDash = false
		case unicode.IsSpace(r) || r == '-' || r == '_' || unicode.IsPunct(r) || unicode.IsSymbol(r):
			if !lastDash && builder.Len() > 0 {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}

	slug := strings.Trim(builder.String(), "-")
	if slug == "" {
		return "item"
	}

	return slug
}

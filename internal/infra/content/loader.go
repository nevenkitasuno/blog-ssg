package content

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/gomarkdown/markdown"

	"github.com/nevenkitasuno/blog-ssg/internal/domain"
)

var entryPattern = regexp.MustCompile(`^(\d{4})\s+(\d{2})\s+(.+)$`)

type Loader struct {
	contentDir string
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

	title := strings.TrimSpace(matches[3])
	if title == "" {
		return domain.Entry{}, false, nil
	}

	files, err := os.ReadDir(filepath.Join(l.contentDir, topicName, dirName))
	if err != nil {
		return domain.Entry{}, false, fmt.Errorf("read entry %q: %w", dirName, err)
	}

	entry := domain.Entry{
		Name:  dirName,
		Slug:  slugify(dirName),
		Year:  year,
		Month: month,
		Title: title,
		Pages: make([]domain.Page, 0, len(files)),
	}

	for _, file := range files {
		if file.IsDir() {
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

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return domain.Page{}, false, nil
	}

	return domain.Page{
		Number: number,
		File: domain.MarkdownFile{
			Path: path,
			Body: trimmed,
			HTML: template.HTML(markdown.ToHTML(body, nil, nil)),
		},
	}, true, nil
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

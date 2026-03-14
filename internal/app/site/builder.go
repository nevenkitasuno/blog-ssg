package site

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/nevenkitasuno/blog-ssg/internal/domain"
	"github.com/nevenkitasuno/blog-ssg/internal/infra/state"
)

type Loader interface {
	Load() (domain.Blog, error)
}

type Renderer interface {
	RenderIndex(data any) ([]byte, error)
	RenderTopic(data any) ([]byte, error)
	RenderPage(data any) ([]byte, error)
	Digest() string
}

type ManifestStore interface {
	Load() (state.Manifest, error)
	Save(manifest state.Manifest) error
}

type Builder struct {
	loader        Loader
	renderer      Renderer
	manifestStore ManifestStore
	outputDir     string
}

type Result struct {
	Generated int
	Skipped   int
	Removed   int
}

type generatedFile struct {
	path        string
	fingerprint string
	render      func() ([]byte, error)
}

type indexTopicView struct {
	Name string
	URL  string
}

type indexView struct {
	Topics []indexTopicView
	CSS    string
	Icon   string
}

type topicYearView struct {
	Year    int
	Entries []topicEntryView
}

type topicEntryView struct {
	Label string
	URL   string
}

type topicView struct {
	Name  string
	Years []topicYearView
	Home  string
	CSS   string
	Icon  string
}

type pageView struct {
	TopicName   string
	TopicURL    string
	EntryName   string
	PageNumber  int
	ContentHTML template.HTML
	NextLabel   string
	NextURL     string
	HomeURL     string
	CSS         string
	Icon        string
}

func NewBuilder(loader Loader, renderer Renderer, manifestStore ManifestStore, outputDir string) *Builder {
	return &Builder{
		loader:        loader,
		renderer:      renderer,
		manifestStore: manifestStore,
		outputDir:     outputDir,
	}
}

func (b *Builder) Build() (Result, error) {
	blog, err := b.loader.Load()
	if err != nil {
		return Result{}, err
	}

	manifest, err := b.manifestStore.Load()
	if err != nil {
		return Result{}, err
	}

	desired, err := b.plan(blog)
	if err != nil {
		return Result{}, err
	}

	nextManifest := state.Manifest{Files: make(map[string]string, len(desired))}
	result := Result{}

	for _, file := range desired {
		nextManifest.Files[file.path] = file.fingerprint

		if manifest.Files[file.path] == file.fingerprint && fileExists(file.path) {
			result.Skipped++
			continue
		}

		content, err := file.render()
		if err != nil {
			return Result{}, fmt.Errorf("render %q: %w", file.path, err)
		}

		if err := os.MkdirAll(filepath.Dir(file.path), 0o755); err != nil {
			return Result{}, fmt.Errorf("create output directory: %w", err)
		}
		if err := os.WriteFile(file.path, content, 0o644); err != nil {
			return Result{}, fmt.Errorf("write output file %q: %w", file.path, err)
		}

		result.Generated++
	}

	for path := range manifest.Files {
		if _, ok := nextManifest.Files[path]; ok {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return Result{}, fmt.Errorf("remove stale file %q: %w", path, err)
		}
		removeEmptyParents(filepath.Dir(path), b.outputDir)
		result.Removed++
	}

	if err := b.manifestStore.Save(nextManifest); err != nil {
		return Result{}, err
	}

	return result, nil
}

func (b *Builder) plan(blog domain.Blog) ([]generatedFile, error) {
	files := make([]generatedFile, 0)
	templateDigest := b.renderer.Digest()

	indexData := indexView{
		Topics: make([]indexTopicView, 0, len(blog.Topics)),
		CSS:    "style.css",
		Icon:   "images/favicon.png",
	}
	for _, topic := range blog.Topics {
		indexData.Topics = append(indexData.Topics, indexTopicView{
			Name: topic.Name,
			URL:  filepath.ToSlash(filepath.Join("topics", topic.Slug, "index.html")),
		})

		topicData := buildTopicView(topic)
		topicPath := filepath.Join(b.outputDir, "topics", topic.Slug, "index.html")
		topicFingerprint := hashStrings(templateDigest, topic.Name)
		for _, entry := range topic.Entries {
			topicFingerprint = hashStrings(topicFingerprint, entry.Name)
			for _, page := range entry.Pages {
				topicFingerprint = hashStrings(topicFingerprint, page.File.Path, page.File.Body)
			}
		}

		topicDataCopy := topicData
		files = append(files, generatedFile{
			path:        topicPath,
			fingerprint: topicFingerprint,
			render: func() ([]byte, error) {
				return b.renderer.RenderTopic(topicDataCopy)
			},
		})

		entryFiles := b.planEntryFiles(topic, filepath.Join(b.outputDir, "topics", topic.Slug))
		files = append(files, entryFiles...)
	}

	indexPath := filepath.Join(b.outputDir, "index.html")
	indexDataCopy := indexData
	indexFingerprint := hashStrings(templateDigest)
	for _, topic := range blog.Topics {
		indexFingerprint = hashStrings(indexFingerprint, topic.Name)
	}

	files = append(files, generatedFile{
		path:        indexPath,
		fingerprint: indexFingerprint,
		render: func() ([]byte, error) {
			return b.renderer.RenderIndex(indexDataCopy)
		},
	})

	slices.SortFunc(files, func(left, right generatedFile) int {
		return strings.Compare(left.path, right.path)
	})

	return files, nil
}

func (b *Builder) planEntryFiles(topic domain.Topic, topicBaseDir string) []generatedFile {
	files := make([]generatedFile, 0)

	for _, entry := range topic.Entries {
		entryDir := filepath.Join(topicBaseDir, entry.Slug)
		for index, page := range entry.Pages {
			pagePath := filepath.Join(entryDir, "index.html")
			if index > 0 {
				pagePath = filepath.Join(entryDir, fmt.Sprintf("%d", page.Number), "index.html")
			}

			view := pageView{
				TopicName:   topic.Name,
				TopicURL:    relativePath(pagePath, filepath.Join(topicBaseDir, "index.html")),
				EntryName:   entry.Name,
				PageNumber:  page.Number,
				ContentHTML: page.File.HTML,
				HomeURL:     relativePath(pagePath, filepath.Join(b.outputDir, "index.html")),
				CSS:         relativePath(pagePath, filepath.Join(b.outputDir, "style.css")),
				Icon:        relativePath(pagePath, filepath.Join(b.outputDir, "images", "favicon.png")),
			}

			if index < len(entry.Pages)-1 {
				next := entry.Pages[index+1]
				nextPath := filepath.Join(entryDir, fmt.Sprintf("%d", next.Number), "index.html")
				view.NextLabel = "Далее"
				view.NextURL = relativePath(pagePath, nextPath)
			} else {
				view.NextLabel = "В начало"
				view.NextURL = relativePath(pagePath, filepath.Join(entryDir, "index.html"))
			}

			viewCopy := view
			fingerprint := hashStrings(
				b.renderer.Digest(),
				topic.Name,
				entry.Name,
				fmt.Sprintf("%d", page.Number),
				page.File.Path,
				page.File.Body,
				view.NextLabel,
				view.NextURL,
			)

			files = append(files, generatedFile{
				path:        pagePath,
				fingerprint: fingerprint,
				render: func() ([]byte, error) {
					return b.renderer.RenderPage(viewCopy)
				},
			})
		}
	}

	return files
}

func buildTopicView(topic domain.Topic) topicView {
	byYear := map[int][]topicEntryView{}
	years := make([]int, 0)

	for _, entry := range topic.Entries {
		if _, ok := byYear[entry.Year]; !ok {
			years = append(years, entry.Year)
		}

		byYear[entry.Year] = append(byYear[entry.Year], topicEntryView{
			Label: fmt.Sprintf("%02d %s", entry.Month, entry.Title),
			URL:   filepath.ToSlash(filepath.Join(entry.Slug, "index.html")),
		})
	}

	slices.SortFunc(years, func(left, right int) int {
		return right - left
	})

	yearViews := make([]topicYearView, 0, len(years))
	for _, year := range years {
		yearViews = append(yearViews, topicYearView{
			Year:    year,
			Entries: byYear[year],
		})
	}

	return topicView{
		Name:  topic.Name,
		Years: yearViews,
		Home:  filepath.ToSlash(filepath.Join("..", "..", "index.html")),
		CSS:   filepath.ToSlash(filepath.Join("..", "..", "style.css")),
		Icon:  filepath.ToSlash(filepath.Join("..", "..", "images", "favicon.png")),
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func removeEmptyParents(dir, stop string) {
	stop = filepath.Clean(stop)
	for {
		if filepath.Clean(dir) == stop {
			return
		}

		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}

		if err := os.Remove(dir); err != nil {
			return
		}

		dir = filepath.Dir(dir)
	}
}

func hashStrings(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		hash.Write([]byte(part))
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func relativePath(fromFile, toFile string) string {
	rel, err := filepath.Rel(filepath.Dir(fromFile), toFile)
	if err != nil {
		return filepath.ToSlash(toFile)
	}
	return filepath.ToSlash(rel)
}

package site

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gomarkdown/markdown"

	"github.com/nevenkitasuno/blog-ssg/internal/domain"
	"github.com/nevenkitasuno/blog-ssg/internal/infra/state"
)

var embeddedImagePattern = regexp.MustCompile(`!\[\[([^\]]+)\]\]`)

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
	Label       string
	URL         string
	PreviewHTML template.HTML
}

type topicTagView struct {
	Name    string
	URL     string
	Current bool
}

type topicLinkView struct {
	Name string
	URL  string
}

type pageTagView struct {
	Name string
	URL  string
}

type topicView struct {
	Name        string
	Description string
	BannerURL   string
	ThemeCSS    template.CSS
	Links       []topicLinkView
	Years       []topicYearView
	Tags        []topicTagView
	Home        string
	ParentName  string
	ParentURL   string
	CSS         string
	Icon        string
}

type pageView struct {
	TopicName   string
	TopicURL    string
	EntryName   string
	EntryTitle  string
	ThemeCSS    template.CSS
	Tags        []pageTagView
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

		topicPath := filepath.Join(b.outputDir, "topics", topic.Slug, "index.html")
		topicData := buildTopicView(topic, topicPath, b.outputDir)
		topicFingerprint := hashStrings(templateDigest, "topic-render-v2", topic.Name, topicThemeFingerprint(topic.Theme))
		for _, link := range topic.Links {
			topicFingerprint = hashStrings(topicFingerprint, link.Label, link.Target, fmt.Sprintf("%t", link.External))
		}
		for _, metaPage := range topic.Meta {
			topicFingerprint = hashStrings(topicFingerprint, metaPage.Name, metaPage.File.Path, metaPage.File.Body)
		}
		for _, asset := range topic.Assets {
			assetFingerprint, err := fileFingerprint(asset.Path)
			if err == nil {
				topicFingerprint = hashStrings(topicFingerprint, assetFingerprint)
			}
		}
		for _, entry := range topic.Entries {
			topicFingerprint = hashStrings(topicFingerprint, entry.Name, strings.Join(entry.Tags, "\x00"))
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

		tagFiles := b.planTagFiles(topic)
		files = append(files, tagFiles...)
		metaFiles := b.planTopicMetaFiles(topic, filepath.Join(b.outputDir, "topics", topic.Slug))
		files = append(files, metaFiles...)
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
				EntryTitle:  entry.Title,
				ThemeCSS:    renderTopicThemeCSS(topic.Theme, relativePath(pagePath, filepath.Join(b.outputDir, topic.Theme.FontFile))),
				Tags:        buildPageTags(pagePath, topicBaseDir, entry),
				PageNumber:  page.Number,
				ContentHTML: renderPageHTML(page.File.Body, pagePath, entryDir),
				HomeURL:     relativePath(pagePath, filepath.Join(b.outputDir, "index.html")),
				CSS:         relativePath(pagePath, filepath.Join(b.outputDir, "style.css")),
				Icon:        relativePath(pagePath, filepath.Join(b.outputDir, "images", "favicon.png")),
			}

			if index < len(entry.Pages)-1 {
				next := entry.Pages[index+1]
				nextPath := filepath.Join(entryDir, fmt.Sprintf("%d", next.Number), "index.html")
				view.NextLabel = "Далее"
				view.NextURL = relativePath(pagePath, nextPath)
			} else if len(entry.Pages) > 1 {
				view.NextLabel = "В начало"
				view.NextURL = relativePath(pagePath, filepath.Join(entryDir, "index.html"))
			}

			viewCopy := view
			fingerprint := hashStrings(
				b.renderer.Digest(),
				"page-render-v3",
				topic.Name,
				topicThemeFingerprint(topic.Theme),
				entry.Name,
				fmt.Sprintf("%d", page.Number),
				page.File.Path,
				page.File.Body,
				strings.Join(entry.Tags, "\x00"),
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

		for _, asset := range entry.Assets {
			assetPath := filepath.Join(entryDir, asset.Name)
			fingerprint, err := fileFingerprint(asset.Path)
			if err != nil {
				continue
			}

			sourcePath := asset.Path
			files = append(files, generatedFile{
				path:        assetPath,
				fingerprint: fingerprint,
				render: func() ([]byte, error) {
					return os.ReadFile(sourcePath)
				},
			})
		}
	}

	return files
}

func (b *Builder) planTagFiles(topic domain.Topic) []generatedFile {
	tagEntries := make(map[string][]domain.Entry)
	tagNames := collectTopicTags(topic)
	files := make([]generatedFile, 0, len(tagNames))

	for _, entry := range topic.Entries {
		for _, tag := range entry.Tags {
			tagEntries[tag] = append(tagEntries[tag], entry)
		}
	}

	for _, tag := range tagNames {
		path := filepath.Join(b.outputDir, "topics", topic.Slug, "tags", slugifySegment(tag), "index.html")
		view := buildArchiveView(
			topic.Name,
			fmt.Sprintf("Тег: %s", tag),
			filepath.ToSlash(filepath.Join("..", "..", "..", "..", "index.html")),
			topic.Name,
			filepath.ToSlash(filepath.Join("..", "..", "index.html")),
			filepath.ToSlash(filepath.Join("..", "..", "..", "..", "style.css")),
			filepath.ToSlash(filepath.Join("..", "..", "..", "..", "images", "favicon.png")),
			topicBannerURL(topic, func(name string) string {
				return filepath.ToSlash(filepath.Join("..", "..", "meta", name))
			}),
			renderTopicThemeCSS(topic.Theme, relativePath(path, filepath.Join(b.outputDir, topic.Theme.FontFile))),
			topic.Links,
			tagEntries[tag],
			tagNames,
			tag,
			filepath.ToSlash(filepath.Join("..", "..", "index.html")),
			func(entry domain.Entry) string {
				return filepath.ToSlash(filepath.Join("..", "..", entry.Slug, "index.html"))
			},
			func(link domain.TopicLink) string {
				if link.External {
					return link.Target
				}
				return filepath.ToSlash(filepath.Join("..", "..", resolveTopicLink(link)))
			},
			func(name string) string {
				return filepath.ToSlash(filepath.Join("..", slugifySegment(name), "index.html"))
			},
		)

		fingerprint := hashStrings(b.renderer.Digest(), "topic-render-v2", topic.Name, tag, topicThemeFingerprint(topic.Theme))
		for _, link := range topic.Links {
			fingerprint = hashStrings(fingerprint, link.Label, link.Target, fmt.Sprintf("%t", link.External))
		}
		for _, metaPage := range topic.Meta {
			fingerprint = hashStrings(fingerprint, metaPage.Name, metaPage.File.Path, metaPage.File.Body)
		}
		for _, entry := range tagEntries[tag] {
			fingerprint = hashStrings(fingerprint, entry.Name, strings.Join(entry.Tags, "\x00"))
			for _, page := range entry.Pages {
				fingerprint = hashStrings(fingerprint, page.File.Path, page.File.Body)
			}
		}

		viewCopy := view
		files = append(files, generatedFile{
			path:        path,
			fingerprint: fingerprint,
			render: func() ([]byte, error) {
				return b.renderer.RenderTopic(viewCopy)
			},
		})
	}

	return files
}

func (b *Builder) planTopicMetaFiles(topic domain.Topic, topicBaseDir string) []generatedFile {
	files := make([]generatedFile, 0, len(topic.Meta)+len(topic.Assets))
	metaOutputDir := filepath.Join(topicBaseDir, "meta")

	for _, metaPage := range topic.Meta {
		pagePath := filepath.Join(metaOutputDir, metaPage.Slug, "index.html")
		view := pageView{
			TopicName:   topic.Name,
			TopicURL:    relativePath(pagePath, filepath.Join(topicBaseDir, "index.html")),
			EntryTitle:  metaPage.Title,
			ThemeCSS:    renderTopicThemeCSS(topic.Theme, relativePath(pagePath, filepath.Join(b.outputDir, topic.Theme.FontFile))),
			ContentHTML: renderPageHTML(metaPage.File.Body, pagePath, metaOutputDir),
			HomeURL:     relativePath(pagePath, filepath.Join(b.outputDir, "index.html")),
			CSS:         relativePath(pagePath, filepath.Join(b.outputDir, "style.css")),
			Icon:        relativePath(pagePath, filepath.Join(b.outputDir, "images", "favicon.png")),
		}

		viewCopy := view
		files = append(files, generatedFile{
			path:        pagePath,
			fingerprint: hashStrings(b.renderer.Digest(), "page-render-v3", topic.Name, topicThemeFingerprint(topic.Theme), metaPage.Name, metaPage.File.Path, metaPage.File.Body),
			render: func() ([]byte, error) {
				return b.renderer.RenderPage(viewCopy)
			},
		})
	}

	for _, asset := range topic.Assets {
		assetPath := filepath.Join(metaOutputDir, asset.Name)
		fingerprint, err := fileFingerprint(asset.Path)
		if err != nil {
			continue
		}

		sourcePath := asset.Path
		files = append(files, generatedFile{
			path:        assetPath,
			fingerprint: fingerprint,
			render: func() ([]byte, error) {
				return os.ReadFile(sourcePath)
			},
		})
	}

	return files
}

func buildTopicView(topic domain.Topic, topicPath, outputDir string) topicView {
	return buildArchiveView(
		topic.Name,
		"",
		filepath.ToSlash(filepath.Join("..", "..", "index.html")),
		"",
		"",
		filepath.ToSlash(filepath.Join("..", "..", "style.css")),
		filepath.ToSlash(filepath.Join("..", "..", "images", "favicon.png")),
		topicBannerURL(topic, func(name string) string {
			return filepath.ToSlash(filepath.Join("meta", name))
		}),
		renderTopicThemeCSS(topic.Theme, relativePath(topicPath, filepath.Join(outputDir, topic.Theme.FontFile))),
		topic.Links,
		topic.Entries,
		collectTopicTags(topic),
		"",
		"",
		func(entry domain.Entry) string {
			return filepath.ToSlash(filepath.Join(entry.Slug, "index.html"))
		},
		func(link domain.TopicLink) string {
			return resolveTopicLink(link)
		},
		func(name string) string {
			return filepath.ToSlash(filepath.Join("tags", slugifySegment(name), "index.html"))
		},
	)
}

func buildArchiveView(
	name string,
	description string,
	home string,
	parentName string,
	parentURL string,
	css string,
	icon string,
	bannerURL string,
	themeCSS template.CSS,
	links []domain.TopicLink,
	entries []domain.Entry,
	allTags []string,
	currentTag string,
	resetURL string,
	entryURL func(domain.Entry) string,
	linkURL func(domain.TopicLink) string,
	tagURL func(string) string,
) topicView {
	byYear := map[int][]topicEntryView{}
	years := make([]int, 0)

	for _, entry := range entries {
		if _, ok := byYear[entry.Year]; !ok {
			years = append(years, entry.Year)
		}

		byYear[entry.Year] = append(byYear[entry.Year], topicEntryView{
			Label:       entryLabel(entry),
			URL:         entryURL(entry),
			PreviewHTML: renderTopicPreviewHTML(entry.Preview),
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

	tagViews := make([]topicTagView, 0, len(allTags))
	if currentTag != "" {
		tagViews = append(tagViews, topicTagView{
			Name: "Сброс",
			URL:  resetURL,
		})
	}
	for _, tag := range allTags {
		tagViews = append(tagViews, topicTagView{
			Name:    tag,
			URL:     tagURL(tag),
			Current: strings.EqualFold(tag, currentTag),
		})
	}

	linkViews := make([]topicLinkView, 0, len(links))
	for _, link := range links {
		linkViews = append(linkViews, topicLinkView{
			Name: link.Label,
			URL:  linkURL(link),
		})
	}

	return topicView{
		Name:        name,
		Description: description,
		BannerURL:   bannerURL,
		ThemeCSS:    themeCSS,
		Links:       linkViews,
		Years:       yearViews,
		Tags:        tagViews,
		Home:        home,
		ParentName:  parentName,
		ParentURL:   parentURL,
		CSS:         css,
		Icon:        icon,
	}
}

func resolveTopicLink(link domain.TopicLink) string {
	if link.External {
		return link.Target
	}

	target := filepath.ToSlash(strings.TrimSpace(link.Target))
	if target == "" {
		return ""
	}

	target = strings.TrimPrefix(target, "./")
	target = strings.TrimPrefix(target, "/")

	if strings.HasSuffix(strings.ToLower(target), ".md") {
		dir := filepath.Dir(target)
		name := strings.TrimSuffix(filepath.Base(target), filepath.Ext(target))
		if number, err := strconv.Atoi(name); err == nil {
			if number <= 1 {
				return filepath.ToSlash(filepath.Join(slugifySegment(dir), "index.html"))
			}
			return filepath.ToSlash(filepath.Join(slugifySegment(dir), fmt.Sprintf("%d", number), "index.html"))
		}

		return filepath.ToSlash(filepath.Join(slugifySegment(dir), slugifySegment(name), "index.html"))
	}

	return filepath.ToSlash(filepath.Join(slugifySegment(target), "index.html"))
}

func entryLabel(entry domain.Entry) string {
	if entry.Day > 0 {
		return fmt.Sprintf("%02d %02d %s", entry.Month, entry.Day, entry.Title)
	}

	return fmt.Sprintf("%02d %s", entry.Month, entry.Title)
}

func renderTopicPreviewHTML(preview string) template.HTML {
	if strings.TrimSpace(preview) == "" {
		return ""
	}

	protected, replacements := protectCustomInlineTags(preview)
	rendered := string(markdown.ToHTML([]byte(protected), nil, nil))
	return template.HTML(restoreCustomInlineTags(rendered, replacements))
}

func renderTopicThemeCSS(theme domain.TopicTheme, fontURL string) template.CSS {
	blocks := make([]string, 0, 2)
	rules := make([]string, 0, 10)
	appendRule := func(variable, value string) {
		if value == "" {
			return
		}
		rules = append(rules, fmt.Sprintf("%s: %s;", variable, value))
	}

	if theme.FontFamily != "" && theme.FontFile != "" && fontURL != "" {
		escapedFamily := strings.ReplaceAll(theme.FontFamily, `"`, `\"`)
		blocks = append(blocks, fmt.Sprintf(`@font-face {
font-family: "%s";
src: url("%s") format("opentype");
font-display: swap;
}`, escapedFamily, fontURL))
		appendRule("--font-topic", fmt.Sprintf(`"%s"`, escapedFamily))
	}

	appendRule("--color-background", theme.Background)
	appendRule("--color-text", theme.Text)
	appendRule("--color-accent", theme.Accent)
	appendRule("--color-heading", theme.Heading)
	appendRule("--color-muted", theme.Muted)
	appendRule("--color-surface", theme.Surface)
	appendRule("--color-border", theme.Border)
	appendRule("--color-code-bg", theme.CodeBG)
	appendRule("--color-code-border", theme.CodeBorder)

	if len(rules) == 0 {
		return template.CSS(strings.Join(blocks, "\n"))
	}

	blocks = append(blocks, ":root {\n"+strings.Join(rules, "\n")+"\n}")
	return template.CSS(strings.Join(blocks, "\n"))
}

func topicThemeFingerprint(theme domain.TopicTheme) string {
	return hashStrings(
		theme.Background,
		theme.FontFamily,
		theme.FontFile,
		theme.Text,
		theme.Accent,
		theme.Heading,
		theme.Muted,
		theme.Surface,
		theme.Border,
		theme.CodeBG,
		theme.CodeBorder,
	)
}

func topicBannerURL(topic domain.Topic, assetURL func(name string) string) string {
	for _, asset := range topic.Assets {
		ext := strings.ToLower(filepath.Ext(asset.Name))
		base := strings.TrimSuffix(strings.ToLower(asset.Name), ext)
		if base != "top_banner" {
			continue
		}
		switch ext {
		case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".avif":
			return assetURL(asset.Name)
		}
	}

	return ""
}

func collectTopicTags(topic domain.Topic) []string {
	seen := map[string]string{}
	for _, entry := range topic.Entries {
		for _, tag := range entry.Tags {
			key := strings.ToLower(tag)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = tag
		}
	}

	tags := make([]string, 0, len(seen))
	for _, tag := range seen {
		tags = append(tags, tag)
	}

	slices.SortFunc(tags, func(left, right string) int {
		return strings.Compare(strings.ToLower(left), strings.ToLower(right))
	})

	return tags
}

func buildPageTags(pagePath, topicBaseDir string, entry domain.Entry) []pageTagView {
	tags := make([]pageTagView, 0, len(entry.Tags))
	for _, tag := range entry.Tags {
		tagPath := filepath.Join(topicBaseDir, "tags", slugifySegment(tag), "index.html")
		tags = append(tags, pageTagView{
			Name: tag,
			URL:  relativePath(pagePath, tagPath),
		})
	}
	return tags
}

func renderPageHTML(markdownBody, pagePath, entryDir string) template.HTML {
	normalized := embeddedImagePattern.ReplaceAllStringFunc(markdownBody, func(match string) string {
		submatches := embeddedImagePattern.FindStringSubmatch(match)
		if len(submatches) != 2 {
			return match
		}

		imageName := strings.TrimSpace(submatches[1])
		if imageName == "" {
			return match
		}

		imagePath := relativePath(pagePath, filepath.Join(entryDir, imageName))
		return fmt.Sprintf("![](%s)", imagePath)
	})

	protected, replacements := protectCustomInlineTags(normalized)
	rendered := string(markdown.ToHTML([]byte(protected), nil, nil))
	return template.HTML(restoreCustomInlineTags(rendered, replacements))
}

func transformCustomInlineTags(input string) string {
	protected, replacements := protectCustomInlineTags(input)
	return restoreCustomInlineTags(protected, replacements)
}

func protectCustomInlineTags(input string) (string, map[string]string) {
	replacements := map[string]string{}
	nextID := 0
	protected, _, _ := parseCustomInlineTags(input, 0, "", true, replacements, &nextID)
	return protected, replacements
}

func parseCustomInlineTags(input string, start int, closingTag string, topLevel bool, replacements map[string]string, nextID *int) (string, int, bool) {
	var builder strings.Builder
	i := start

	for i < len(input) {
		if input[i] == '[' {
			if end := strings.IndexByte(input[i:], ']'); end >= 0 {
				tag := input[i+1 : i+end]
				next := i + end + 1

				if closingTag != "" && tag == "/"+closingTag {
					return builder.String(), next, true
				}

				if isSupportedInlineTag(tag) {
					inner, after, closed := parseCustomInlineTags(input, next, tag, false, replacements, nextID)
					if closed {
						rendered := renderInlineTag(tag, inner)
						if topLevel {
							token := fmt.Sprintf("CUSTOMINLINE%dTOKEN", *nextID)
							*nextID = *nextID + 1
							replacements[token] = rendered
							builder.WriteString(token)
						} else {
							builder.WriteString(rendered)
						}
						i = after
						continue
					}
				}
			}
		}

		_, size := utf8.DecodeRuneInString(input[i:])
		segment := input[i : i+size]
		if topLevel {
			builder.WriteString(segment)
		} else {
			builder.WriteString(template.HTMLEscapeString(segment))
		}
		i += size
	}

	return builder.String(), i, false
}

func restoreCustomInlineTags(input string, replacements map[string]string) string {
	output := input
	for token, replacement := range replacements {
		output = strings.ReplaceAll(output, token, replacement)
	}
	return output
}

func isSupportedInlineTag(tag string) bool {
	switch tag {
	case "font-mahjong-colored", "rot-90":
		return true
	default:
		return false
	}
}

func renderInlineTag(tag, inner string) string {
	switch tag {
	case "font-mahjong-colored":
		return `<span class="font-mahjong-colored"><span class="font-mahjong-colored-run">` + inner + `</span></span>`
	case "rot-90":
		return renderRot90HTML(inner)
	default:
		return inner
	}
}

func renderRot90HTML(inner string) string {
	return `<span class="rot-90"><span class="rot-90-char">` + inner + `</span></span>`
}

func fileFingerprint(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return hashStrings(path, string(content)), nil
}

func slugifySegment(value string) string {
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
		return "tag"
	}

	return slug
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

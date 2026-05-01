package domain

type Blog struct {
	Topics []Topic
}

type Topic struct {
	Name    string
	Slug    string
	Links   []TopicLink
	Theme   TopicTheme
	Meta    []TopicMetaPage
	Assets  []Asset
	Entries []Entry
}

type TopicTheme struct {
	Background string
	FontFamily string
	FontFile   string
	Text       string
	Accent     string
	Heading    string
	Muted      string
	Surface    string
	Border     string
	CodeBG     string
	CodeBorder string
}

type TopicLink struct {
	Label    string
	Target   string
	External bool
}

type TopicMetaPage struct {
	Name  string
	Slug  string
	Title string
	File  MarkdownFile
}

type Entry struct {
	Name    string
	Slug    string
	Year    int
	Month   int
	Day     int
	Title   string
	Preview string
	Tags    []string
	Assets  []Asset
	Pages   []Page
}

type Asset struct {
	Name string
	Path string
}

type Page struct {
	Number int
	File   MarkdownFile
}

type MarkdownFile struct {
	Path string
	Body string
}

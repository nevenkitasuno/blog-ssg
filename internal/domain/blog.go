package domain

type Blog struct {
	Topics []Topic
}

type Topic struct {
	Name    string
	Slug    string
	Entries []Entry
}

type Entry struct {
	Name   string
	Slug   string
	Year   int
	Month  int
	Day    int
	Title  string
	Tags   []string
	Assets []Asset
	Pages  []Page
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

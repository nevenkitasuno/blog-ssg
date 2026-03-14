package domain

import "html/template"

type Blog struct {
	Topics []Topic
}

type Topic struct {
	Name    string
	Slug    string
	Entries []Entry
}

type Entry struct {
	Name  string
	Slug  string
	Year  int
	Month int
	Title string
	Tags  []string
	Pages []Page
}

type Page struct {
	Number int
	File   MarkdownFile
}

type MarkdownFile struct {
	Path string
	Body string
	HTML template.HTML
}

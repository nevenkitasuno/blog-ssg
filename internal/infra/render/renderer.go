package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
)

type Renderer struct {
	indexTemplate *template.Template
	topicTemplate *template.Template
	pageTemplate  *template.Template
	digest        string
}

func NewRenderer(templatesDir string) (*Renderer, error) {
	indexPath := filepath.Join(templatesDir, "index.html")
	topicPath := filepath.Join(templatesDir, "topic.html")
	pagePath := filepath.Join(templatesDir, "page.html")

	indexTemplate, err := template.ParseFiles(indexPath)
	if err != nil {
		return nil, fmt.Errorf("parse index template: %w", err)
	}

	topicTemplate, err := template.ParseFiles(topicPath)
	if err != nil {
		return nil, fmt.Errorf("parse topic template: %w", err)
	}

	pageTemplate, err := template.ParseFiles(pagePath)
	if err != nil {
		return nil, fmt.Errorf("parse page template: %w", err)
	}

	digest, err := digestTemplates(indexPath, topicPath, pagePath)
	if err != nil {
		return nil, err
	}

	return &Renderer{
		indexTemplate: indexTemplate,
		topicTemplate: topicTemplate,
		pageTemplate:  pageTemplate,
		digest:        digest,
	}, nil
}

func (r *Renderer) RenderIndex(data any) ([]byte, error) {
	return execute(r.indexTemplate, data)
}

func (r *Renderer) RenderTopic(data any) ([]byte, error) {
	return execute(r.topicTemplate, data)
}

func (r *Renderer) RenderPage(data any) ([]byte, error) {
	return execute(r.pageTemplate, data)
}

func (r *Renderer) Digest() string {
	return r.digest
}

func execute(tpl *template.Template, data any) ([]byte, error) {
	var buffer bytes.Buffer
	if err := tpl.Execute(&buffer, data); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func digestTemplates(paths ...string) (string, error) {
	hash := sha256.New()
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read template %q: %w", path, err)
		}
		hash.Write(content)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

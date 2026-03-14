package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/nevenkitasuno/blog-ssg/internal/app/site"
	"github.com/nevenkitasuno/blog-ssg/internal/infra/content"
	"github.com/nevenkitasuno/blog-ssg/internal/infra/render"
	"github.com/nevenkitasuno/blog-ssg/internal/infra/state"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("ssg: %v", err)
	}
}

func run() error {
	contentDir := flag.String("content", "content", "path to content directory")
	templatesDir := flag.String("templates", "templates", "path to templates directory")
	outputDir := flag.String("output", "output", "path to output directory")
	flag.Parse()

	loader := content.NewLoader(*contentDir)
	renderer, err := render.NewRenderer(*templatesDir)
	if err != nil {
		return err
	}

	manifestStore := state.NewManifestStore(*outputDir)
	builder := site.NewBuilder(loader, renderer, manifestStore, *outputDir)

	result, err := builder.Build()
	if err != nil {
		return err
	}

	fmt.Fprintf(
		os.Stdout,
		"generated %d file(s), skipped %d unchanged file(s), removed %d stale file(s)\n",
		result.Generated,
		result.Skipped,
		result.Removed,
	)

	return nil
}

package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const manifestName = ".ssg-manifest.json"

type Manifest struct {
	Files map[string]string `json:"files"`
}

type ManifestStore struct {
	path string
}

func NewManifestStore(outputDir string) *ManifestStore {
	return &ManifestStore{
		path: filepath.Join(outputDir, manifestName),
	}
}

func (s *ManifestStore) Load() (Manifest, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return Manifest{Files: map[string]string{}}, nil
		}
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	if manifest.Files == nil {
		manifest.Files = map[string]string{}
	}

	return manifest, nil
}

func (s *ManifestStore) Save(manifest Manifest) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create manifest directory: %w", err)
	}

	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}

	if err := os.WriteFile(s.path, append(content, '\n'), 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}

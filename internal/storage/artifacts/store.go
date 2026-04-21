package artifacts

import (
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

type Store struct {
	root string
}

func New(root string) *Store {
	if strings.TrimSpace(root) == "" {
		root = filepath.Join(".data", "artifacts")
	}
	return &Store{root: root}
}

func (s *Store) Save(storageKey string, src io.Reader) (int64, error) {
	path := filepath.Join(s.root, storageKey)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return 0, fmt.Errorf("mkdir artifact dir: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return 0, fmt.Errorf("create artifact file: %w", err)
	}
	defer file.Close()

	n, err := io.Copy(file, src)
	if err != nil {
		return 0, fmt.Errorf("save artifact file: %w", err)
	}
	return n, nil
}

func (s *Store) Open(storageKey string) (io.ReadCloser, error) {
	path := filepath.Join(s.root, storageKey)
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open artifact file: %w", err)
	}
	return file, nil
}

func (s *Store) Delete(storageKey string) error {
	path := filepath.Join(s.root, storageKey)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete artifact file: %w", err)
	}
	return nil
}

func DetectContentType(name, fallback string) string {
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		return "application/octet-stream"
	}
	if guessed := mime.TypeByExtension(ext); guessed != "" {
		return guessed
	}
	return "application/octet-stream"
}

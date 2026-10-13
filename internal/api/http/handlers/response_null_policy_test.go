package handlers

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Clients rely on every response key being present, absent values as an explicit null; omitempty
// would silently drop the key instead.
func TestResponseShapesNeverOmitEmptyFields(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(source), "omitempty") {
			t.Errorf("%s uses omitempty; answer absent values as a nil pointer that encodes as null", path)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walking the handler sources: %v", err)
	}
}

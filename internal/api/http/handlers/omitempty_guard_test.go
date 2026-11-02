package handlers

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Responses answer a missing value with an explicit null, which omitempty would silently drop.
// The scan is textual, so it also catches omitempty on a request struct; spell it out there if needed.
func TestHandlersDeclareNoOmitemptyFields(t *testing.T) {
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
			t.Errorf("%s uses omitempty; make the field a pointer so it encodes as null", path)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walking the handlers tree: %v", err)
	}
}

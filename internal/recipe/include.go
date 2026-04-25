package recipe

import (
	"os"
	"path/filepath"
	"strings"
)

const grlxExt = ".grlx"

// ResolveInclude resolves an include directive to a file path.
// The include value can be:
//   - dot-prefixed (e.g. ".foo") — resolved relative to the including file's directory
//   - plain name (e.g. "foo") — resolved relative to basePath (the recipe root)
//
// Resolution order: check <name>/init.grlx first, then <name>.grlx.
// Dots in non-relative names are converted to path separators (e.g. "a.b" → "a/b").
func ResolveInclude(basePath, currentFile, include string) (string, bool) {
	var searchDir string

	if strings.HasPrefix(include, ".") {
		name := strings.TrimPrefix(include, ".")
		searchDir = filepath.Dir(currentFile)
		return resolveRecipePath(searchDir, name)
	}

	name := strings.ReplaceAll(include, ".", string(filepath.Separator))
	searchDir = basePath
	return resolveRecipePath(searchDir, name)
}

func resolveRecipePath(dir, name string) (string, bool) {
	initPath := filepath.Join(dir, name, "init"+grlxExt)
	if _, err := os.Stat(initPath); err == nil {
		return initPath, true
	}

	filePath := filepath.Join(dir, name+grlxExt)
	if _, err := os.Stat(filePath); err == nil {
		return filePath, true
	}

	return "", false
}

package figtree

import (
	"os"
	"path"
	"path/filepath"
	"strings"
)

func FindParentPaths(homedir, cwd, fileName string) []string {
	paths := make([]string, 0)

	// special case if homedir is not in current path then check there anyway
	if !strings.HasPrefix(cwd, homedir) {
		file := path.Join(homedir, fileName)
		if _, err := os.Stat(file); err == nil {
			paths = append(paths, filepath.FromSlash(file))
		}
	}

	var dir string
	for _, part := range strings.Split(cwd, string(os.PathSeparator)) {
		if part == "" && dir == "" {
			dir = "/"
		} else {
			dir = path.Join(dir, part)
		}
		file := path.Join(dir, fileName)
		if _, err := os.Stat(file); err == nil {
			paths = append(paths, filepath.FromSlash(file))
		}
	}
	return paths
}

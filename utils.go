package figtree

import (
	"fmt"
	"os"
	"strings"
)

func findParentPaths(fileName string) []string {
	cwd, _ := os.Getwd()

	paths := make([]string, 0)

	// special case if homedir is not in current path then check there anyway
	homedir := os.Getenv("HOME")
	if !strings.HasPrefix(cwd, homedir) {
		file := fmt.Sprintf("%s/%s", homedir, fileName)
		if _, err := os.Stat(file); err == nil {
			paths = append(paths, file)
		}
	}

	var dir string
	for _, part := range strings.Split(cwd, string(os.PathSeparator)) {
		if dir == "/" {
			dir = fmt.Sprintf("/%s", part)
		} else {
			dir = fmt.Sprintf("%s/%s", dir, part)
		}
		file := fmt.Sprintf("%s/%s", dir, fileName)
		if _, err := os.Stat(file); err == nil {
			paths = append(paths, file)
		}
	}
	return paths
}

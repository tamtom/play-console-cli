package update

import (
	"os"
	"strings"
)

// removeStaleBackups removes the "<name>.old-*" files that an earlier update
// left in the root. On Windows, the running image stays locked until the
// process exits, so an update cannot remove its own backup.
func removeStaleBackups(root *os.Root, name string) {
	dir, err := root.Open(".")
	if err != nil {
		return
	}
	entries, err := dir.ReadDir(-1)
	_ = dir.Close()
	if err != nil {
		return
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), name+".old-") && entry.Type().IsRegular() {
			_ = root.Remove(entry.Name())
		}
	}
}

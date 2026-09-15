package export

import "os"

// WriteFile writes data to path with the given permission bits, creating parent
// directories as needed.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	if perm == 0 {
		perm = 0o644
	}
	if dir := dirOf(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, perm)
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == os.PathSeparator {
			return path[:i]
		}
	}
	return ""
}

package configtui

import "path/filepath"

func dirOf(p string) string { return filepath.Dir(p) }

func joinPath(dir, rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Join(dir, rel)
}

package config

import "os"

func writeFileImpl(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}

package compilers

import (
	"os"
	"path/filepath"
)

func mkdirWithFiles(name string, files map[string]string) error {
	var (
		key, val string
		err      error
	)

	err = os.MkdirAll(name, 0700)
	if err != nil {
		return err
	}

	for key, val = range files {
		err = os.WriteFile(filepath.Join(name, key), []byte(val), 0400)
		if err != nil {
			return err
		}
	}

	return nil
}

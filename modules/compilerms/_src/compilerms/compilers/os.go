package compilers

import (
	"os"
	"path/filepath"
)

func mkdirTemp(pattern string) (string, error) {
	var (
		tmpdir string
		err    error
	)

	tmpdir, err = os.MkdirTemp(os.TempDir(), pattern)
	if err != nil {
		return "", err
	}

	return tmpdir, nil
}

func mkdirFiles(name string, files map[string]string) error {
	var (
		key, val string
		err      error
	)

	err = os.Mkdir(name, 0700)
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

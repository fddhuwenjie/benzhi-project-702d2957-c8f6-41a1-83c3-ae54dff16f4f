package store

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
)

var safeID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{2,79}$`)

func validateID(id string) error {
	if !safeID.MatchString(id) {
		return errors.New("编号格式无效")
	}
	return nil
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".pending-*")
	if err != nil {
		return err
	}
	name := f.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(name)
		}
	}()
	if err = f.Chmod(mode); err != nil {
		_ = f.Close()
		return err
	}
	if _, err = f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(name, path); err != nil {
		return err
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	if err = d.Sync(); err != nil {
		return err
	}
	ok = true
	return nil
}

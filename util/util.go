package util

import (
	"io"
	"os"
)

func IsDir(path string) bool {
	stat := getFileStat(path)
	if stat == nil {
		return false
	}

	return stat.Mode().IsDir()
}

func IsFile(path string) bool {
	stat := getFileStat(path)
	if stat == nil {
		return false
	}

	return stat.Mode().IsRegular()
}

func getFileStat(path string) os.FileInfo {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil
	}

	return stat
}

/**
 * @see: https://gist.github.com/elazarl/5507969
 */
func Copy(dst, src string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	// no need to check errors on read only file, we already got everything
	// we need from the filesystem, so nothing can go wrong now.
	defer s.Close()
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}
	return d.Close()
}

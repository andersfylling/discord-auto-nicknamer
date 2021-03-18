package nicknamer

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type Storage interface {
	Save([]byte) error
	Load() ([]byte, error)
}

type FileStorage struct {
	FileName string
	DirPath  string
}

func (fs *FileStorage) Directory() string {
	if fs.DirPath == "" {
		return "."
	}
	return strings.TrimSuffix(fs.DirPath, "/")
}

func (fs *FileStorage) Save(b []byte) error {
	dir := fs.Directory()
	filename := fs.FileName
	path := dir + "/" + filename

	if _, err := os.Stat(path); os.IsNotExist(err) {
		Log.Errorf("unable to create old version, path does not exists: %v", err)
	} else {
		// save old version
		lastInd := strings.LastIndex(filename, ".")
		oldFilename := filename[:lastInd] + ".old." + filename[lastInd+1:]
		if err := os.Rename(path, dir+"/"+oldFilename); err != nil {
			return fmt.Errorf("unable to create old version: %w", err)
		}
	}

	Log.Infof("writing to %s: %s", path, string(b))
	return ioutil.WriteFile(path, b, 0644)
}

func (fs *FileStorage) Load() ([]byte, error) {
	dir := fs.Directory()
	filename := fs.FileName
	path := dir + "/" + filename
	return ioutil.ReadFile(path)
}

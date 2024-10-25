package file

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
)

// write body to file at path
func Write(path string, body []byte) error {
	// create the file
	f, err := os.Create(path)
	if err != nil {
		// fmt.Println(err)
		return err
	}
	// close the file with defer
	defer f.Close()

	//write directly into file
	_, err = f.Write(body)
	return err
}

// copy source to target
func Copy(sourcePath string, targetPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	dir, _ := path.Split(targetPath)
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}
	target, err := os.OpenFile(targetPath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer target.Close()

	_, err = io.Copy(target, source)
	if err != nil {
		return err
	}
	return nil
}

// reads directory returns lines or error
func ReadDir(root string) ([]string, error) {
	var files []string
	fileInfo, err := os.ReadDir(root)
	if err != nil {
		return files, err
	}
	for _, file := range fileInfo {
		files = append(files, file.Name())
	}
	return files, nil
}

// path exists?
func Exists(path string) bool {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		// path/to/whatever does not exist
		return false
	}
	return true
}

func ReadFileAsBytes(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func FileExists(path string) (exists bool) {
	_, err := os.Stat(path)
	return !errors.Is(err, fs.ErrNotExist)
}

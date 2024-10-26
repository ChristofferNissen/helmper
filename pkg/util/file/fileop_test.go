package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrite(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "testfile.txt")
	body := []byte("Hello, World!")

	err := Write(filePath, body)
	assert.NoError(t, err)

	// Verify the file content
	content, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, body, content)
}

func TestWriteError(t *testing.T) {
	err := Write("/invalid/path/testfile.txt", []byte("Hello, World!"))
	assert.Error(t, err)
}

func TestCopy(t *testing.T) {
	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "source.txt")
	targetPath := filepath.Join(tempDir, "target.txt")
	body := []byte("Hello, World!")

	// Write source file
	err := os.WriteFile(sourcePath, body, 0644)
	assert.NoError(t, err)

	err = Copy(sourcePath, targetPath)
	assert.NoError(t, err)

	// Verify the target file content
	content, err := os.ReadFile(targetPath)
	assert.NoError(t, err)
	assert.Equal(t, body, content)
}

func TestCopyError(t *testing.T) {
	err := Copy("/invalid/path/source.txt", "/invalid/path/target.txt")
	assert.Error(t, err)
}

func TestReadDir(t *testing.T) {
	tempDir := t.TempDir()

	files := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, f := range files {
		err := os.WriteFile(filepath.Join(tempDir, f), []byte("content"), 0644)
		assert.NoError(t, err)
	}

	result, err := ReadDir(tempDir)
	assert.NoError(t, err)
	assert.ElementsMatch(t, files, result)
}

func TestReadDirError(t *testing.T) {
	_, err := ReadDir("/invalid/path")
	assert.Error(t, err)
}

func TestExists(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "testfile.txt")

	// File should not exist initially
	assert.False(t, Exists(filePath))

	// Create the file
	err := os.WriteFile(filePath, []byte("content"), 0644)
	assert.NoError(t, err)

	// Now the file should exist
	assert.True(t, Exists(filePath))
}

func TestReadFileAsBytes(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "testfile.txt")
	body := []byte("Hello, World!")

	// Write the file
	err := os.WriteFile(filePath, body, 0644)
	assert.NoError(t, err)

	content, err := ReadFileAsBytes(filePath)
	assert.NoError(t, err)
	assert.Equal(t, body, content)
}

func TestReadFileAsBytesError(t *testing.T) {
	_, err := ReadFileAsBytes("/invalid/path/testfile.txt")
	assert.Error(t, err)
}

func TestFileExists(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "testfile.txt")

	// File should not exist initially
	assert.False(t, FileExists(filePath))

	// Create the file
	err := os.WriteFile(filePath, []byte("content"), 0644)
	assert.NoError(t, err)

	// Now the file should exist
	assert.True(t, FileExists(filePath))
}

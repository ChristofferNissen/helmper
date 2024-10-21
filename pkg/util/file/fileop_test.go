package file

import (
	"os"
	"testing"
)

func TestFileExists(t *testing.T) {
	filename := "testfile"
	_, err := os.Create(filename)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer os.Remove(filename)

	if !FileExists(filename) {
		t.Errorf("expected file to exist")
	}

	if FileExists("nonexistentfile") {
		t.Errorf("expected file not to exist")
	}
}

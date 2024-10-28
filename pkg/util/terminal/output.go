package terminal

import (
	"bytes"
	"os"
	"sync"
)

func CaptureOutput(f func() error) (string, error) {
	// Create a pipe
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}

	// Save the original stdout and stderr
	origStdout := os.Stdout
	origStderr := os.Stderr
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()

	// Redirect stdout and stderr to the pipe
	os.Stdout = w
	os.Stderr = w

	// Create a buffer to capture the output
	var buf bytes.Buffer
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		_, _ = buf.ReadFrom(r)
	}()

	// Call the function and capture the error
	err = f()

	// Close the writer and wait for the reader to finish
	w.Close()
	wg.Wait()

	return buf.String(), err
}

package terminal

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCaptureOutput(t *testing.T) {
	// Test successful function call
	successfulFunc := func() error {
		fmt.Println("Hello, World!")
		return nil
	}

	output, err := CaptureOutput(successfulFunc)
	assert.NoError(t, err)
	assert.Equal(t, "Hello, World!\n", output)

	// Test function call with error
	errorFunc := func() error {
		fmt.Println("An error occurred")
		return errors.New("error")
	}

	output, err = CaptureOutput(errorFunc)
	assert.Error(t, err)
	assert.Equal(t, "An error occurred\n", output)
}

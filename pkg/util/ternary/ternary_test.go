package ternary

import (
	"testing"
)

const (
	TRUE  bool = true
	FALSE bool = false
)

func TestTernaryTrue(t *testing.T) {
	expected := TRUE
	actual := Ternary(TRUE, TRUE, FALSE)

	if actual != expected {
		t.Errorf("want '%t' got '%t'", expected, actual)
	}
}

func TestTernaryFalse(t *testing.T) {
	expected := FALSE
	actual := Ternary(FALSE, TRUE, FALSE)

	if actual != expected {
		t.Errorf("want '%t' got '%t'", expected, actual)
	}
}

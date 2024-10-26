package terminal

import (
	"testing"

	"github.com/enescakir/emoji"
	"github.com/stretchr/testify/assert"
)

func TestPrintGreen(t *testing.T) {
	// No assertion required as we're just printing to console
	PrintGreen("Test Green")
}

func TestPrintRed(t *testing.T) {
	// No assertion required as we're just printing to console
	PrintRed("Test Red")
}

func TestPrintYellow(t *testing.T) {
	// No assertion required as we're just printing to console
	PrintYellow("Test Yellow")
}

func TestLogYellow(t *testing.T) {
	// No assertion required as we're just logging to console
	LogYellow("Test Log Yellow")
}

func TestGetCheckMarkEmoji(t *testing.T) {
	e := GetCheckMarkEmoji()
	assert.Equal(t, e, emoji.CheckMarkButton.String())
}

func TestGetWarningEmoji(t *testing.T) {
	e := GetWarningEmoji()
	assert.Equal(t, e, emoji.Warning.String())
}

func TestGetErrorEmoji(t *testing.T) {
	e := GetErrorEmoji()
	assert.Equal(t, e, emoji.CrossMark.String())
}

func TestGetDetectiveEmoji(t *testing.T) {
	e := GetDetectiveEmoji()
	assert.Equal(t, e, emoji.Detective.String())
}

func TestGetHourglassEmoji(t *testing.T) {
	e := GetHourglassEmoji()
	assert.Equal(t, e, emoji.HourglassNotDone.String())
}

func TestStatusEmoji(t *testing.T) {
	checkMark := StatusEmoji(true)
	assert.Equal(t, checkMark, GetCheckMarkEmoji())

	errorMark := StatusEmoji(false)
	assert.Equal(t, errorMark, GetErrorEmoji())
}

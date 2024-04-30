package terminal

import (
	"fmt"
	"log"

	"github.com/enescakir/emoji"
)

// Colored prints

func PrintGreen(text string) {
	colorReset := "\033[0m"
	colorGreen := "\033[32m"

	fmt.Printf("%s%s%s\n", string(colorGreen), text, string(colorReset))
}

func PrintRed(text string) {
	colorReset := "\033[0m"
	colorRed := "\033[31m"

	fmt.Printf("%s%s%s\n", string(colorRed), text, string(colorReset))
}

func PrintYellow(text string) {
	colorReset := "\033[0m"
	colorYellow := "\033[33m"

	fmt.Printf("%s%s%s\n", string(colorYellow), text, string(colorReset))
}

func LogYellow(text string) {
	colorReset := "\033[0m"
	colorYellow := "\033[33m"

	log.Printf("%s%s%s\n", string(colorYellow), text, string(colorReset))
}

// Emojis

func GetCheckMarkEmoji() string {
	return emoji.CheckMarkButton.String()
}

func GetWarningEmoji() string {
	return emoji.Warning.String()
}

func GetErrorEmoji() string {
	return emoji.CrossMark.String()
}

func GetDetectiveEmoji() string {
	return emoji.Detective.String()
}

func GetHourglassEmoji() string {
	return emoji.HourglassNotDone.String()
}

func StatusEmoji(b bool) string {
	if b {
		return GetCheckMarkEmoji()
	}
	return GetErrorEmoji()
}

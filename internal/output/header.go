package output

import (
	"fmt"

	"github.com/ChristofferNissen/helmper/pkg/util/terminal"
	"github.com/common-nighthawk/go-figure"
)

func Header(version, commit, date string) {
	myFigure := figure.NewFigure("helmper", "rectangles", true)
	myFigure.Print()
	terminal.PrintYellow(fmt.Sprintf("version %s (commit %s, built at %s)\n", version, commit, date))
}

// Copyright by sirupsen
//
// file taken from https://github.com/sirupsen/logrus
// add the support for additional fixed fields.
// Because of usage of many unecessarily provide fields,
// types and functions, all the stuff has to be copied
// to be extended.
//

package logrusfmt

import (
	"io"
	"os"

	"github.com/mattn/go-isatty"
)

func checkIfTerminal(w io.Writer) bool {
	switch v := w.(type) {
	case *os.File:
		return isatty.IsTerminal(v.Fd())
	default:
		return false
	}
}

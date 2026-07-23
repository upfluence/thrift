package gocodegen

import (
	"bytes"
	"os/exec"
)

// GoFmt runs gofmt on src and returns the formatted output. If gofmt is not
// available or fails, the original src is returned unchanged.
func GoFmt(src []byte) []byte {
	gofmtPath, err := exec.LookPath("gofmt")

	if err != nil {
		return src
	}

	var out bytes.Buffer

	cmd := exec.Command(gofmtPath) //nolint:gosec
	cmd.Stdin = bytes.NewReader(src)
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return src
	}

	return out.Bytes()
}

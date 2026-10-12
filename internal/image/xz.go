package image

import (
	"fmt"
	"io"
	"os/exec"
)

type xzExecReader struct {
	cmd    *exec.Cmd
	pipe   io.ReadCloser
	done   chan error
}

func newXZExecReader(r io.Reader) (io.ReadCloser, error) {
	cmd := exec.Command("xz", "-d", "-c")
	cmd.Stdin = r

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start xz: %w (is xz installed?)", err)
	}

	return &xzExecReader{
		cmd:  cmd,
		pipe: pipe,
	}, nil
}

func (x *xzExecReader) Read(p []byte) (int, error) {
	return x.pipe.Read(p)
}

func (x *xzExecReader) Close() error {
	x.pipe.Close()
	return x.cmd.Wait()
}

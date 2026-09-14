package compilers

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"

	"github.com/nanmu42/limitio"
	"golang.org/x/sys/unix"
)

func run(
	ctx context.Context,
	stdin io.Reader,
	name string,
	args ...string,
) (Output, error) {
	var (
		builder     strings.Builder
		limitWriter *limitio.Writer
		cmd         *exec.Cmd
		output      Output
		ok          bool
		exitErr     *exec.ExitError
		err         error
	)

	limitWriter = limitio.NewWriter(&builder, 4096, false)

	cmd = exec.CommandContext(ctx, name, args...)
	cmd.Stdin = stdin
	cmd.Stdout = limitWriter
	cmd.Stderr = limitWriter

	cmd.SysProcAttr = &unix.SysProcAttr{
		Pdeathsig: unix.SIGKILL,
		Setpgid:   true,
	}

	err = cmd.Run()
	output.Output = builder.String()
	if err != nil {
		output.Status = 1

		exitErr, ok = errors.AsType[*exec.ExitError](err)
		if ok {
			output.Status = exitErr.ExitCode()
		}

		return output, err
	}

	return output, nil
}

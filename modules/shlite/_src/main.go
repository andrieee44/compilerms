package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

func expand(args []string) ([]string, error) {
	var (
		newArgs, matches []string
		arg              string
		err              error
	)

	newArgs = make([]string, 0, len(args))

	for _, arg = range args {
		matches, err = filepath.Glob(arg)
		if err != nil {
			return nil, err
		}

		if len(matches) == 0 {
			newArgs = append(newArgs, arg)

			continue
		}

		newArgs = append(newArgs, matches...)
	}

	return newArgs, nil
}

func newEnv(keys string) []string {
	var (
		env []string
		i   int
	)

	env = strings.Split(keys, ":")

	for i = range env {
		if env[i] != "" {
			env[i] = fmt.Sprintf("%s=%s", env[i], os.Getenv(env[i]))
		}
	}

	return env
}

func run() error {
	var (
		cmd     *exec.Cmd
		args    []string
		ok      bool
		exitErr *exec.ExitError
		err     error
	)

	if len(os.Args) < 3 {
		return errors.New("missing arguments")
	}

	args, err = expand(os.Args[2:])
	if err != nil {
		return err
	}

	cmd = exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.WaitDelay = 100 * time.Millisecond
	cmd.SysProcAttr = &unix.SysProcAttr{Pdeathsig: unix.SIGKILL}
	cmd.Env = newEnv(os.Args[1])

	err = cmd.Run()
	if err != nil {
		exitErr, ok = errors.AsType[*exec.ExitError](err)
		if ok {
			os.Exit(exitErr.ExitCode())
		}

		return err
	}

	return nil
}

func main() {
	var err error

	err = run()
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			`shlite: %v

Usage:   shlite <ENV_KEY>[:<ENV_KEY>...] <PROGRAM> [ARG]...
Example: shlite PATH echo *
         shlite PATH echo \*
         shlite PATH:HOME:PWD env
`, err)

		os.Exit(1)
	}
}

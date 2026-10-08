package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func inheritEnv(keys ...string) []string {
	var (
		envs []string
		key  string
	)

	envs = make([]string, 0, len(keys))

	for _, key = range keys {
		envs = append(envs, fmt.Sprintf("%s=%s", key, os.Getenv(key)))
	}

	return envs
}

func run() error {
	var (
		newArgs, matches []string
		arg              string
		cmd              *exec.Cmd
		ok               bool
		exitErr          *exec.ExitError
		err              error
	)

	if len(os.Args) < 2 {
		return errors.New("no program specified")
	}

	newArgs = make([]string, 0, len(os.Args)-1)

	for _, arg = range os.Args[1:] {
		matches, err = filepath.Glob(arg)
		if err != nil {
			return err
		}

		if len(matches) == 0 {
			newArgs = append(newArgs, arg)

			continue
		}

		newArgs = append(newArgs, matches...)
	}

	cmd = exec.Command(newArgs[0], newArgs[1:]...)
	cmd.Env = inheritEnv("PATH")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

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
		fmt.Fprintf(os.Stderr, `shlite: %v

Usage: shlite <PROGRAM> [ARG]...
`, err)

		os.Exit(1)
	}
}

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"time"

	"github.com/coder/websocket"
	"github.com/nanmu42/limitio"
	"golang.org/x/sys/unix"
)

type LanguageStep struct {
	Key   string
	Value ConfigStep
}

type Sandbox struct {
	lang          *Language
	cfg           *Config
	files         map[string]string
	rodir, rundir string
}

var (
	ErrBadRequest error = errors.New("bad request")
	ErrStep       error = errors.New("step error")
)

func (sb *Sandbox) Config() *ConfigLanguage {
	return sb.lang.Config()
}

func (sb *Sandbox) Steps() YAMLOrderedMap[ConfigStep] {
	return sb.Config().Steps
}

func (sb *Sandbox) SandboxSystemdRunFlags() []string {
	return []string{
		"-p", fmt.Sprintf("BindPaths=%s:/rundir", sb.rundir),
		"-p", fmt.Sprintf("BindReadOnlyPaths=%s:/rodir %s",
			sb.rodir,
			sb.cfg.Global.Commands.Shlite,
		),
	}
}

func (sb *Sandbox) WebSocketSystemdRunFlags(enableWs bool) []string {
	if enableWs {
		return sb.cfg.Global.WebSocket.SystemdRun.Strings()
	}

	return []string{}
}

func (sb *Sandbox) ApplyCmdOpts(cmd *exec.Cmd) *exec.Cmd {
	cmd.WaitDelay = 2 * time.Second

	cmd.Cancel = func() error {
		return unix.Kill(-cmd.Process.Pid, unix.SIGKILL)
	}

	cmd.SysProcAttr = &unix.SysProcAttr{
		Pdeathsig: unix.SIGKILL,
		Setpgid:   true,
	}

	return cmd
}

func (sb *Sandbox) StepCmd(
	ctx context.Context,
	step *LanguageStep,
	enableWs bool,
) *exec.Cmd {
	return sb.ApplyCmdOpts(
		exec.CommandContext(
			ctx,
			sb.cfg.Global.Commands.SystemdRun,
			slices.Concat(
				sb.cfg.Global.SystemdRun.Strings(),
				step.Value.SystemdRun.Strings(),
				sb.SandboxSystemdRunFlags(),
				sb.WebSocketSystemdRunFlags(enableWs),
				[]string{
					"--",
					sb.cfg.Global.Commands.Shlite,
					step.Value.Command.Name,
				},
				step.Value.Command.Flags.Strings(),
				step.Value.Command.Args,
			)...,
		),
	)
}

func (sb *Sandbox) DebugStepCmd(
	ctx context.Context,
	step *LanguageStep,
) *exec.Cmd {
	return sb.ApplyCmdOpts(
		exec.CommandContext(
			ctx,
			sb.cfg.Global.Commands.SystemdRun,
			slices.Concat(
				sb.cfg.Global.SystemdRun.Strings(),
				step.Value.SystemdRun.Strings(),
				sb.SandboxSystemdRunFlags(),
				sb.cfg.Global.Strace.SystemdRun.Strings(),
				[]string{
					"-p", fmt.Sprintf(
						"BindReadOnlyPaths=%s",
						sb.cfg.Global.Commands.Strace,
					),
					"--",
					sb.cfg.Global.Commands.Strace,
				},
				sb.cfg.Global.Strace.Flags.Strings(),
				[]string{
					"--",
					sb.cfg.Global.Commands.Shlite,
					step.Value.Command.Name,
				},
				step.Value.Command.Flags.Strings(),
				step.Value.Command.Args,
			)...,
		),
	)
}

func (sb *Sandbox) Compile(
	ctx context.Context,
	output io.Writer,
	debug bool,
) error {
	var (
		step       LanguageStep
		stepOutput io.Writer
		cmd        *exec.Cmd
		err        error
	)

	for _, step = range sb.Steps()[:len(sb.Steps())-1] {
		stepOutput = limitio.NewWriter(
			output,
			sb.cfg.Global.Language.StepMaxBytesOutput,
			false,
		)

		cmd = sb.StepCmd(ctx, &step, false)

		if debug {
			cmd = sb.DebugStepCmd(ctx, &step)
		}

		cmd.Stdout = stepOutput
		cmd.Stderr = stepOutput

		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("%s %w: %w", step.Key, ErrStep, err)
		}
	}

	return nil
}

func (sb *Sandbox) Program(
	ctx context.Context,
	stdin io.Reader,
	output io.Writer,
	debug bool,
) error {
	var (
		step LanguageStep
		cmd  *exec.Cmd
		err  error
	)

	stdin = limitio.NewReader(
		stdin,
		sb.cfg.Global.Language.StepMaxBytesStdin,
		false,
	)

	output = limitio.NewWriter(
		output,
		sb.cfg.Global.Language.StepMaxBytesOutput,
		false,
	)

	step = sb.Steps()[len(sb.Steps())-1]
	cmd = sb.StepCmd(ctx, &step, false)

	if debug {
		cmd = sb.DebugStepCmd(ctx, &step)
	}

	cmd.Stdin = stdin
	cmd.Stdout = output
	cmd.Stderr = output

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("%s %w: %w", step.Key, ErrStep, err)
	}

	return nil
}

func (sb *Sandbox) ProgramWs(ctx context.Context, ws *websocket.Conn) error {
	var (
		stdinPr, outputPr *io.PipeReader
		stdinPw, outputPw *io.PipeWriter
		stdin             io.Reader
		output            io.Writer
		step              LanguageStep
		cmd               *exec.Cmd
		err               error
	)

	stdinPr, stdinPw = io.Pipe()
	defer stdinPr.Close()
	defer stdinPw.Close()

	outputPr, outputPw = io.Pipe()
	defer outputPr.Close()
	defer outputPw.Close()

	stdin = limitio.NewReader(
		stdinPr,
		sb.cfg.Global.Language.StepMaxBytesStdin,
		false,
	)

	output = limitio.NewWriter(
		outputPw,
		sb.cfg.Global.Language.StepMaxBytesOutput,
		false,
	)

	step = sb.Steps()[len(sb.Steps())-1]
	cmd = sb.StepCmd(ctx, &step, true)
	cmd.Stdin = stdin
	cmd.Stdout = output
	cmd.Stderr = output

	go func() {
		var (
			buf []byte
			err error
		)

		for {
			_, buf, err = ws.Read(ctx)
			if err != nil {
				return
			}

			_, err = stdinPw.Write(buf)
			if err != nil {
				return
			}
		}
	}()

	go func() {
		var (
			n   int
			buf []byte
			err error
		)

		buf = make([]byte, 4096)

		for {
			n, err = outputPr.Read(buf)
			if err != nil {
				return
			}

			err = ws.Write(ctx, websocket.MessageBinary, buf[:n])
			if err != nil {
				return
			}
		}
	}()

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("%s %w: %w", step.Key, ErrStep, err)
	}

	return nil
}

func (sb *Sandbox) Close() error {
	var err error

	err = os.RemoveAll(sb.rodir)
	if err != nil {
		os.RemoveAll(sb.rundir)

		return err
	}

	return os.RemoveAll(sb.rundir)
}

func (sb *Sandbox) MatchFiles() error {
	var (
		file, pattern string
		good, ok      bool
		err           error
	)

	for file = range sb.files {
		good = false

		for _, pattern = range sb.Config().Files.Whitelist {
			ok, err = filepath.Match(pattern, file)
			if err != nil {
				return err
			}

			if ok {
				good = true
			}
		}

		if !good {
			return fmt.Errorf("%w: %s: invalid file", ErrBadRequest, file)
		}
	}

	return nil
}

func (sb *Sandbox) PopulateRodir() error {
	var (
		root          *os.Root
		file, content string
		err           error
	)

	root, err = os.OpenRoot(sb.rodir)
	if err != nil {
		return err
	}

	for file, content = range sb.files {
		err = root.MkdirAll(filepath.Dir(file), 0700)
		if err != nil {
			return err
		}

		err = root.WriteFile(file, []byte(content), 0400)
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *Language) Sandbox(files map[string]string) (*Sandbox, error) {
	var (
		sb  Sandbox
		err error
	)

	sb.lang = l
	sb.cfg = l.cfg
	sb.files = files

	err = sb.MatchFiles()
	if err != nil {
		return nil, err
	}

	sb.rodir, err = os.MkdirTemp(
		os.TempDir(),
		fmt.Sprintf("compilerms-%s-readonly-*", l.Name()),
	)
	if err != nil {
		return nil, err
	}

	sb.rundir, err = os.MkdirTemp(
		os.TempDir(),
		fmt.Sprintf("compilerms-%s-runtime-*", l.Name()),
	)
	if err != nil {
		os.RemoveAll(sb.rodir)

		return nil, err
	}

	err = sb.PopulateRodir()
	if err != nil {
		os.RemoveAll(sb.rodir)
		os.RemoveAll(sb.rundir)

		return nil, err
	}

	return &sb, nil
}

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/nanmu42/limitio"
	"golang.org/x/sys/unix"
)

func newLimitReader(r io.Reader, limit int) io.Reader {
	if r == nil {
		return nil
	}

	return limitio.NewReader(r, limit, false)
}

func newLimitWriter(w io.Writer, limit int) io.Writer {
	if w == nil {
		return nil
	}

	return limitio.NewWriter(w, limit, false)
}

func pipe(dst io.Writer, src io.Reader) {
	if dst == nil || src == nil {
		return
	}

	_, _ = io.Copy(dst, src)
}

func checkFiles(files map[string]string, allowed []string) error {
	var (
		fileName, pattern string
		goodFile, ok      bool
		err               error
	)

	for fileName = range files {
		goodFile = false

		for _, pattern = range allowed {
			ok, err = filepath.Match(pattern, fileName)
			if err != nil {
				return err
			}

			if ok {
				goodFile = true

				break
			}
		}

		if !goodFile {
			return newStatusError(
				http.StatusUnprocessableEntity,
				websocket.StatusPolicyViolation,
				fmt.Errorf("%s: bad fileName", fileName),
			)
		}
	}

	return nil
}

func mkTmpFiles(langCfg *langConfig, files map[string]string) (string, error) {
	var (
		tmpDir, dirName, fileName, fileContent string
		root                                   *os.Root
		err                                    error
	)

	err = checkFiles(files, langCfg.AllowedFiles)
	if err != nil {
		return "", err
	}

	tmpDir, err = os.MkdirTemp("", "compilerms-*")
	if err != nil {
		return "", err
	}

	defer func() {
		if err != nil {
			_ = os.RemoveAll(tmpDir)
		}
	}()

	root, err = os.OpenRoot(tmpDir)
	if err != nil {
		return "", err
	}

	defer func() { _ = root.Close() }()

	for _, dirName = range []string{"rodir", "rundir"} {
		err = root.Mkdir(dirName, 0700)
		if err != nil {
			return "", err
		}
	}

	for fileName, fileContent = range files {
		err = root.MkdirAll(
			filepath.Join("rodir", filepath.Dir(fileName)),
			0700,
		)
		if err != nil {
			return "", err
		}

		err = root.WriteFile(
			filepath.Join("rodir", fileName),
			[]byte(fileContent),
			0400,
		)
		if err != nil {
			return "", err
		}
	}

	return tmpDir, nil
}

func newCommand(ctx context.Context, name string, arg ...string) *exec.Cmd {
	var cmd *exec.Cmd

	cmd = exec.CommandContext(ctx, name, arg...)
	cmd.WaitDelay = 100 * time.Millisecond

	cmd.Cancel = func() error {
		return unix.Kill(-cmd.Process.Pid, unix.SIGKILL)
	}

	cmd.SysProcAttr = &unix.SysProcAttr{
		Pdeathsig: unix.SIGKILL,
		Setpgid:   true,
	}

	return cmd
}

func newArgs(templateArgs []string, tmpDir string) []string {
	var (
		replacer *strings.Replacer
		args     []string
		arg      string
	)

	replacer = strings.NewReplacer(
		"{{ COMPILERMS_RODIR }}", filepath.Join(tmpDir, "rodir"),
		"{{ COMPILERMS_RUNDIR }}", filepath.Join(tmpDir, "rundir"),
	)

	args = make([]string, 0, len(templateArgs))
	for _, arg = range templateArgs {
		args = append(args, replacer.Replace(arg))
	}

	return args
}

type runStepRequest struct {
	stepCfg *stepConfig
	tmpDir  string
	input   io.Reader
	output  io.Writer
	debug   bool
}

type runStepResponse struct {
	status int
}

func runStep(
	ctx context.Context,
	req *runStepRequest,
) (*runStepResponse, error) {
	var (
		args    []string
		cmd     *exec.Cmd
		stdin   io.WriteCloser
		output  io.ReadCloser
		ok      bool
		exitErr *exec.ExitError
		err     error
	)

	err = sem.Acquire(ctx, 1)
	if err != nil {
		return nil, err
	}

	defer sem.Release(1)

	if !req.debug {
		args = newArgs(req.stepCfg.Args.Prod, req.tmpDir)
	} else {
		args = newArgs(req.stepCfg.Args.Debug, req.tmpDir)
	}

	cmd = newCommand(ctx, req.stepCfg.Name, args...)

	stdin, err = cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	output, err = cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	cmd.Stderr = cmd.Stdout

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	go pipe(stdin, newLimitReader(req.input, req.stepCfg.MaxBytesInput))
	go pipe(newLimitWriter(req.output, req.stepCfg.MaxBytesOutput), output)

	err = cmd.Wait()
	if err != nil {
		exitErr, ok = errors.AsType[*exec.ExitError](err)
		if ok {
			return &runStepResponse{status: exitErr.ExitCode()}, nil
		}

		return nil, err
	}

	return &runStepResponse{status: 0}, nil
}

type runVersionsRequest struct {
	cfg *config
}

type runVersionsResponse struct {
	Versions map[string]map[string]string `json:"versions"`
}

func runVersions(
	ctx context.Context,
	req *runVersionsRequest,
) (*runVersionsResponse, error) {
	var (
		resp              *runVersionsResponse
		langName, cmdName string
		langCfg           *langConfig
		cmdCfg            *cmdConfig
		output            []byte
		err               error
	)

	resp = &runVersionsResponse{Versions: make(map[string]map[string]string)}

	for langName, langCfg = range req.cfg.Languages {
		resp.Versions[langName] = make(map[string]string)

		for cmdName, cmdCfg = range langCfg.Versions {
			output, err = newCommand(
				ctx,
				cmdCfg.Name,
				cmdCfg.Args...,
			).CombinedOutput()
			if err != nil {
				return nil, err
			}

			resp.Versions[langName][cmdName] = string(output)
		}
	}

	return resp, nil
}

type runLiveRequest struct {
	Files   map[string]string `json:"files"`
	langCfg *langConfig
	stepCfg *stepConfig
	input   io.Reader
	output  io.Writer
	debug   bool
}

type runLiveResponse struct {
	Name   string `json:"name"`
	Status int    `json:"status"`
}

func runLive(
	ctx context.Context,
	req *runLiveRequest,
) (*runLiveResponse, error) {
	var (
		tmpDir      string
		runStepResp *runStepResponse
		err         error
	)

	tmpDir, err = mkTmpFiles(req.langCfg, req.Files)
	if err != nil {
		return nil, err
	}

	defer func() { _ = os.RemoveAll(tmpDir) }()

	if req.langCfg.Compiler != nil {
		runStepResp, err = runStep(ctx, &runStepRequest{
			stepCfg: req.langCfg.Compiler,
			tmpDir:  tmpDir,
			input:   nil,
			output:  req.output,
			debug:   req.debug,
		})
		if err != nil {
			return nil, err
		}

		if runStepResp.status != 0 {
			return &runLiveResponse{
				Name:   req.langCfg.Compiler.DisplayName,
				Status: runStepResp.status,
			}, nil
		}
	}

	runStepResp, err = runStep(ctx, &runStepRequest{
		stepCfg: req.stepCfg,
		tmpDir:  tmpDir,
		input:   req.input,
		output:  req.output,
		debug:   req.debug,
	})
	if err != nil {
		return nil, err
	}

	return &runLiveResponse{
		Name:   req.stepCfg.DisplayName,
		Status: runStepResp.status,
	}, nil
}

type runBatchRequest struct {
	Files   map[string]string `json:"files"`
	Input   string            `json:"input"`
	langCfg *langConfig
	debug   bool
}

type runBatchResponse struct {
	Name   string `json:"name"`
	Output string `json:"output"`
	Status int    `json:"status"`
}

func runBatch(
	ctx context.Context,
	req *runBatchRequest,
) (*runBatchResponse, error) {
	var (
		output bytes.Buffer
		resp   *runLiveResponse
		err    error
	)

	resp, err = runLive(ctx, &runLiveRequest{
		Files:   req.Files,
		langCfg: req.langCfg,
		stepCfg: req.langCfg.Program.Batch,
		input:   strings.NewReader(req.Input),
		output:  &output,
		debug:   req.debug,
	})
	if err != nil {
		return nil, err
	}

	return &runBatchResponse{
		Name:   resp.Name,
		Output: output.String(),
		Status: resp.Status,
	}, nil
}

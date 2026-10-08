package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/coder/websocket"
)

type Language struct {
	name string
	cfg  *Config
}

type LanguageRequest struct {
	Files map[string]string `json:"files"`
	Stdin string            `json:"stdin"`
}

func (l *Language) Name() string {
	return l.name
}

func (l *Language) Config() *ConfigLanguage {
	var langCfg ConfigLanguage

	langCfg = l.cfg.Languages[l.Name()]

	return &langCfg
}

func (l *Language) StatusCode(err error) int {
	var (
		exitErr *exec.ExitError
		ok      bool
	)

	if err == nil {
		return 0
	}

	exitErr, ok = errors.AsType[*exec.ExitError](err)
	if ok {
		return exitErr.ExitCode()
	}

	return 1
}

func (l *Language) RunTest(test *ConfigTest, debug bool) error {
	var (
		sb  *Sandbox
		err error
	)

	sb, err = l.Sandbox(test.Files)
	if err != nil {
		return err
	}

	defer sb.Close()

	err = sb.Compile(context.Background(), os.Stderr, debug)
	if err != nil {
		return err
	}

	return sb.Program(
		context.Background(),
		strings.NewReader(test.Stdin),
		os.Stderr,
		debug,
	)
}

func (l *Language) RunTests() error {
	var (
		name          string
		test          ConfigTest
		debugErr, err error
	)

	for name, test = range l.Config().Tests {
		slog.Info("Running Test", "language", l.Name(), "test", name)

		err = l.RunTest(&test, false)
		if err == nil {
			slog.Info("Successful Test")

			continue
		}

		slog.Error("Failed Test", "error", err)
		slog.Info("Debugging Test")

		debugErr = l.RunTest(&test, true)
		if debugErr != nil {
			slog.Error("Failed Debug Test", "error", debugErr)

			return debugErr
		}

		slog.Info("Successful Debug Test")

		return err
	}

	return nil
}

func (l *Language) WriteResponse(
	w http.ResponseWriter,
	r *http.Request,
	output string,
	stepErr error,
) {
	var (
		stepErrStr string
		err        error
	)

	if stepErr != nil {
		stepErrStr = stepErr.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(struct {
		Output string `json:"output"`
		Status int    `json:"status"`
		Error  string `json:"error,omitempty"`
	}{
		Output: output,
		Status: l.StatusCode(stepErr),
		Error:  stepErrStr,
	})
	if err != nil {
		slog.Error(r.URL.String(), "method", r.Method, "error", err)
	}
}

func (l *Language) CompileHandler(
	w http.ResponseWriter,
	r *http.Request,
) (*LanguageRequest, *Sandbox) {
	var (
		decoder *json.Decoder
		req     LanguageRequest
		sb      *Sandbox
		output  bytes.Buffer
		err     error
	)

	decoder = json.NewDecoder(http.MaxBytesReader(
		w,
		r.Body,
		int64(l.cfg.Global.HTTPServer.MaxBytesReader),
	))

	decoder.DisallowUnknownFields()

	err = decoder.Decode(&req)
	if err != nil {
		slog.Error(r.URL.String(), "method", r.Method, "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)

		return nil, nil
	}

	sb, err = l.Sandbox(req.Files)
	switch {
	case errors.Is(err, ErrBadRequest):
		http.Error(w, err.Error(), http.StatusBadRequest)

		return nil, nil
	case err != nil:
		slog.Error(r.URL.String(), "method", r.Method, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)

		return nil, nil
	}

	err = sb.Compile(r.Context(), &output, false)
	switch {
	case errors.Is(err, ErrStep):
		l.WriteResponse(w, r, output.String(), err)
		sb.Close()

		return nil, nil
	case err != nil:
		slog.Error(r.URL.String(), "method", r.Method, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		sb.Close()

		return nil, nil
	}

	return &req, sb
}

func (l *Language) Handler(w http.ResponseWriter, r *http.Request) {
	var (
		req    *LanguageRequest
		sb     *Sandbox
		output bytes.Buffer
		err    error
	)

	req, sb = l.CompileHandler(w, r)
	if req == nil || sb == nil {
		return
	}

	defer sb.Close()

	err = sb.Program(
		r.Context(),
		strings.NewReader(req.Stdin),
		&output,
		false,
	)
	if err != nil && !errors.Is(err, ErrStep) {
		slog.Error(r.URL.String(), "method", r.Method, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)

		return
	}

	l.WriteResponse(w, r, output.String(), err)
}

func (l *Language) HandlerWS(w http.ResponseWriter, r *http.Request) {
	var (
		sb     *Sandbox
		ws     *websocket.Conn
		ctx    context.Context
		cancel context.CancelFunc
		err    error
	)

	_, sb = l.CompileHandler(w, r)
	if sb == nil {
		return
	}

	sb, err = l.Sandbox(nil)
	if err != nil {
		panic(err)
	}

	defer sb.Close()

	ws, err = websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		slog.Error(r.URL.String(), "method", r.Method, "error", err)

		return
	}

	defer ws.CloseNow()

	ws.SetReadLimit(int64(l.cfg.Global.WebSocket.ReadLimit))

	ctx, cancel = context.WithTimeout(
		context.Background(),
		l.cfg.Global.WebSocket.Timeout,
	)
	defer cancel()

	err = sb.ProgramWs(ctx, ws)
	switch {
	case errors.Is(err, ErrStep):
		ws.Write(ctx, websocket.MessageBinary, []byte(err.Error()))
	case err != nil:
		slog.Error(r.URL.String(), "method", r.Method, "error", err)
		ws.Close(websocket.StatusInternalError, "Internal Error")

		return
	}

	ws.Close(websocket.StatusNormalClosure, "Normal Closure")
}

func NewLanguage(name string, cfg *Config) *Language {
	return &Language{
		name: name,
		cfg:  cfg,
	}
}

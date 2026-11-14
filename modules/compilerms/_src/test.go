package main

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/cli/go-gh/v2/pkg/jsonpretty"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"golang.org/x/sync/errgroup"
)

func compareStatus(got, exp int) error {
	if got == exp {
		return nil
	}

	return fmt.Errorf(
		"expected %d (%s), got %d (%s)",
		exp, http.StatusText(exp),
		got, http.StatusText(got),
	)
}

func printJSON(value any) error {
	var (
		reader io.Reader
		buf    []byte
		err    error
	)

	switch value := value.(type) {
	case io.Reader:
		reader = value
	default:
		buf, err = json.Marshal(value)
		if err != nil {
			return err
		}

		reader = bytes.NewReader(buf)
	}

	return jsonpretty.Format(os.Stderr, reader, "  ", true)
}

type testRequest struct {
	client   *http.Client
	url      string
	langName string
	testName string
	langCfg  *langConfig
	testCfg  *testConfig
}

type testDebugFn func(req *testRequest) (any, error)

func debugLive(req *testRequest) (any, error) {
	return runLive(context.Background(), &runLiveRequest{
		Files:   req.testCfg.Files,
		langCfg: req.langCfg,
		stepCfg: req.langCfg.Program.Live,
		input:   strings.NewReader(req.testCfg.Input),
		output:  os.Stderr,
		debug:   true,
	})
}

func debugBatch(req *testRequest) (any, error) {
	return runBatch(context.Background(), &runBatchRequest{
		Files:   req.testCfg.Files,
		Input:   req.testCfg.Input,
		langCfg: req.langCfg,
		debug:   true,
	})
}

func testLive(ctx context.Context, req *testRequest) error {
	var (
		conn    *websocket.Conn
		builder strings.Builder
		typ     websocket.MessageType
		data    []byte
		resp    runLiveResponse
		err     error
	)

	conn, _, err = websocket.Dial(ctx, req.url+"/"+req.langName, nil)
	if err != nil {
		return err
	}

	defer func() { _ = conn.CloseNow() }()

	err = wsjson.Write(ctx, conn, runLiveRequest{Files: req.testCfg.Files})
	if err != nil {
		return err
	}

	err = conn.Write(ctx, websocket.MessageBinary, []byte(req.testCfg.Input))
	if err != nil {
		return err
	}

loop:
	for {
		typ, data, err = conn.Read(ctx)
		if err != nil {
			return err
		}

		switch typ {
		case websocket.MessageBinary:
			builder.Write(data)
		case websocket.MessageText:
			err = json.Unmarshal(data, &resp)
			if err != nil {
				return err
			}

			break loop
		}
	}

	if resp.Status != 0 {
		return fmt.Errorf("exit status %d", resp.Status)
	}

	if builder.String() != req.testCfg.Output {
		return fmt.Errorf(
			"expected %q, got %q",
			req.testCfg.Output, builder.String(),
		)
	}

	_ = conn.Close(
		websocket.StatusNormalClosure,
		websocket.StatusNormalClosure.String(),
	)

	return nil

}

func testBatch(ctx context.Context, req *testRequest) error {
	var (
		buf           bytes.Buffer
		httpReq       *http.Request
		httpResp      *http.Response
		runResp       runBatchResponse
		debugErr, err error
	)

	err = json.MarshalWrite(&buf, runBatchRequest{
		Files: req.testCfg.Files,
		Input: req.testCfg.Input,
	})
	if err != nil {
		return err
	}

	httpReq, err = http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		req.url+"/"+req.langName,
		&buf,
	)
	if err != nil {
		return err
	}

	httpResp, err = req.client.Do(httpReq)
	if err != nil {
		return err
	}

	defer func() { _ = httpResp.Body.Close() }()

	err = compareStatus(httpResp.StatusCode, http.StatusOK)
	if err != nil {
		return err
	}

	err = json.UnmarshalRead(httpResp.Body, &runResp)
	if err != nil {
		return err
	}

	if runResp.Status != 0 {
		debugErr = printJSON(runResp)
		if debugErr != nil {
			return debugErr
		}

		return fmt.Errorf("exit status %d", runResp.Status)
	}

	if runResp.Output != req.testCfg.Output {
		return fmt.Errorf(
			"expected %q, got %q",
			req.testCfg.Output, runResp.Output,
		)
	}

	return nil
}

func testVersions(client *http.Client, url string) error {
	var (
		resp *http.Response
		err  error
	)

	resp, err = client.Get(url)
	if err != nil {
		return err
	}

	defer func() { _ = resp.Body.Close() }()

	return compareStatus(resp.StatusCode, http.StatusOK)
}

func newTestFn(
	ctx context.Context,
	req *testRequest,
	testFn func(context.Context, *testRequest) error,
	debugFn testDebugFn,
) func() error {
	return func() error {
		var err error

		err = testFn(ctx, req)
		if err != nil {
			return newTestError(req, debugFn, err)
		}

		return nil
	}
}

func testLanguages(
	languagesCfg map[string]*langConfig,
	client *http.Client,
	url string,
) error {
	var (
		eg            *errgroup.Group
		ctx           context.Context
		resp          any
		ok            bool
		testErr       *testError
		debugErr, err error
	)

	slog.Info("running tests concurrently")
	eg, ctx = errgroup.WithContext(context.Background())
	eg.Go(func() error { return testVersions(client, url) })

	for langName, langCfg := range languagesCfg {
		for testName, testCfg := range langCfg.Tests.Live {
			req := &testRequest{
				client:   client,
				url:      url,
				langName: langName,
				testName: testName,
				langCfg:  langCfg,
				testCfg:  testCfg,
			}

			eg.Go(newTestFn(ctx, req, testLive, debugLive))
		}

		for testName, testCfg := range langCfg.Tests.Batch {
			req := &testRequest{
				client:   client,
				url:      url,
				langName: langName,
				testName: testName,
				langCfg:  langCfg,
				testCfg:  testCfg,
			}

			eg.Go(newTestFn(ctx, req, testBatch, debugBatch))
		}
	}

	err = eg.Wait()
	if err != nil {
		testErr, ok = err.(*testError)
		if !ok {
			return err
		}

		slog.Error(
			"failed test, running debug",
			"lang", testErr.req.langName,
			"test", testErr.req.testName,
			"error", err,
		)

		resp, debugErr = testErr.debugFn(testErr.req)
		if debugErr != nil {
			slog.Error(
				"failed debug",
				"lang", testErr.req.langName,
				"test", testErr.req.testName,
				"error", debugErr,
			)

			return debugErr
		}

		debugErr = printJSON(resp)
		if debugErr != nil {
			slog.Error(
				"failed to print debug output",
				"lang", testErr.req.langName,
				"test", testErr.req.testName,
				"error", debugErr,
			)

			return debugErr
		}

		return err
	}

	slog.Info("all tests passed")

	return nil
}

func testServer(cfg *config) error {
	var testSrv *httptest.Server

	testSrv = httptest.NewUnstartedServer(nil)
	testSrv.Config = newServer(cfg)
	testSrv.Start()
	defer testSrv.Close()

	return testLanguages(cfg.Languages, testSrv.Client(), testSrv.URL)
}

package main

import "github.com/coder/websocket"

type statusError struct {
	httpStatus int
	wsStatus   websocket.StatusCode
	err        error
}

func (e *statusError) Error() string { return e.err.Error() }
func (e *statusError) Unwrap() error { return e.err }

func newStatusError(
	httpStatus int,
	wsStatus websocket.StatusCode,
	err error,
) *statusError {
	return &statusError{
		httpStatus: httpStatus,
		wsStatus:   wsStatus,
		err:        err,
	}
}

type testError struct {
	req     *testRequest
	debugFn testDebugFn
	err     error
}

func (e *testError) Error() string { return e.err.Error() }
func (e *testError) Unwrap() error { return e.err }

func newTestError(req *testRequest, debugFn testDebugFn, err error) *testError {
	return &testError{
		req:     req,
		debugFn: debugFn,
		err:     err,
	}
}

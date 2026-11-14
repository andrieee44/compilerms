package main

import (
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
)

func logHandlerError(r *http.Request, err error) {
	slog.Error(r.Method+" "+r.URL.String(), "error", err)
}

func newJSONHandler[T any](
	fn func(http.ResponseWriter, *http.Request) (*T, error),
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			resp      *T
			status    int
			ok        bool
			statusErr *statusError
			err       error
		)

		resp, err = fn(w, r)
		if err != nil {
			status = http.StatusInternalServerError

			statusErr, ok = errors.AsType[*statusError](err)
			if ok {
				status = statusErr.httpStatus
			}

			http.Error(w, http.StatusText(status), status)
			logHandlerError(r, err)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err = json.MarshalWrite(w, resp)
		if err != nil {
			logHandlerError(r, err)
		}
	})
}

func newWSHandler(
	wsCfg *websocketConfig,
	fn func(*websocket.Conn) error,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			conn      *websocket.Conn
			status    websocket.StatusCode
			ok        bool
			statusErr *statusError
			err       error
		)

		conn, err = websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns:  wsCfg.OriginPatterns,
			CompressionMode: websocket.CompressionContextTakeover,
		})
		if err != nil {
			logHandlerError(r, err)

			return
		}

		defer func() { _ = conn.CloseNow() }()

		status = websocket.StatusNormalClosure

		err = fn(conn)
		if err != nil {
			status = websocket.StatusInternalError

			statusErr, ok = errors.AsType[*statusError](err)
			if ok {
				status = statusErr.wsStatus
			}

			logHandlerError(r, err)
		}

		err = conn.Close(status, status.String())
		if err != nil {
			logHandlerError(r, err)
		}
	})
}

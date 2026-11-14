package main

import (
	"context"
	"encoding/json/v2"
	"net"
	"net/http"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func newVersionsHandler(cfg *config) http.Handler {
	return newJSONHandler(func(
		_ http.ResponseWriter,
		r *http.Request,
	) (*runVersionsResponse, error) {
		return runVersions(r.Context(), &runVersionsRequest{cfg: cfg})
	})
}

func newLiveHandler(wsCfg *websocketConfig, langCfg *langConfig) http.Handler {
	return newWSHandler(wsCfg, func(conn *websocket.Conn) error {
		var (
			ctx     context.Context
			cancel  context.CancelFunc
			netConn net.Conn
			req     runLiveRequest
			resp    *runLiveResponse
			err     error
		)

		conn.SetReadLimit(wsCfg.ReadLimit)

		ctx, cancel = context.WithTimeout(
			context.Background(),
			wsCfg.Timeout.Duration,
		)

		defer cancel()

		err = wsjson.Read(ctx, conn, &req)
		if err != nil {
			return newStatusError(
				http.StatusBadRequest,
				websocket.StatusInvalidFramePayloadData,
				err,
			)
		}

		netConn = websocket.NetConn(ctx, conn, websocket.MessageBinary)

		req.langCfg = langCfg
		req.stepCfg = langCfg.Program.Live
		req.input = netConn
		req.output = netConn
		req.debug = false

		resp, err = runLive(ctx, &req)
		if err != nil {
			return err
		}

		return wsjson.Write(ctx, conn, resp)
	})
}

func newBatchHandler(httpCfg *httpConfig, langCfg *langConfig) http.Handler {
	return newJSONHandler(func(
		w http.ResponseWriter,
		r *http.Request,
	) (*runBatchResponse, error) {
		var (
			req runBatchRequest
			err error
		)

		r.Body = http.MaxBytesReader(w, r.Body, httpCfg.MaxBytesReader)
		defer func() { _ = r.Body.Close() }()

		err = json.UnmarshalRead(r.Body, &req, json.RejectUnknownMembers(true))
		if err != nil {
			return nil, newStatusError(
				http.StatusBadRequest,
				websocket.StatusInvalidFramePayloadData,
				err,
			)
		}

		req.langCfg = langCfg
		req.debug = false

		return runBatch(r.Context(), &req)
	})
}

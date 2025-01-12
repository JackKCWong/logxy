package internal

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var _ http.Handler = (*SingleHostProxy)(nil)

type SingleHostProxy struct {
	Target *url.URL
	Client *http.Client
}

// ServeHTTP implements http.Handler.
func (h *SingleHostProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodConnect {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()
	reqMetrics := zerolog.Dict()
	reqMetrics.Time("arrived_at", start)

	respMetrics := zerolog.Dict()

	logevt := log.Info().
		Str("id", generateID())

	latencies := zerolog.Arr()

	defer func() {
		logevt.
			Dict("req", reqMetrics).
			Dict("resp", respMetrics).
			Array("latency", latencies).
			Send()
	}()

	recordLatency := func(name string) {
		latencies.Dict(zerolog.Dict().Dur(name, time.Since(start)))
	}

	clientTraceCtx := httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{
		DNSStart: func(di httptrace.DNSStartInfo) {
			recordLatency("dns_start")
		},
		DNSDone: func(di httptrace.DNSDoneInfo) {
			recordLatency("dns_done")
		},
		TLSHandshakeStart: func() {
			recordLatency("tls_start")
		},
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			recordLatency("tls_done")
		},
		GetConn: func(hostPort string) {
			recordLatency("get_conn")
		},
		ConnectStart: func(network, addr string) {
			recordLatency("conn_start")
		},
		ConnectDone: func(network, addr string, err error) {
			recordLatency("conn_done")
		},
		GotConn: func(gci httptrace.GotConnInfo) {
			recordLatency("got_conn")
		},
		WroteHeaders: func() {
			recordLatency("wrote_req_headers")
		},
		WroteRequest: func(wri httptrace.WroteRequestInfo) {
			recordLatency("wrote_req_body")
		},
		GotFirstResponseByte: func() {
			recordLatency("TTFB_resp")
		},
	})

	reqTracker := &ReaderTracker{Reader: req.Body}
	req.URL.Scheme = h.Target.Scheme
	req.URL.Host = h.Target.Host
	req.Body = reqTracker
	req.RequestURI = ""
	req = req.WithContext(clientTraceCtx)

	resp, err := h.Client.Do(req)

	recordLatency("wrote_resp")

	reqMetrics.
		Str("method", req.Method).
		Str("url", req.URL.String()).
		Int("body_bytes", reqTracker.BytesRead()).
		Int("body_read_bps", reqTracker.BytesReadPerSecond()).
		Dur("body_read_ms", reqTracker.Duration())

	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(err.Error()))
		logevt.Err(err)
		return
	}

	defer resp.Body.Close()
	respTracker := &ReaderTracker{Reader: resp.Body}
	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}

	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, respTracker)
	if err != nil {
		logevt.Err(err)
	}

	respMetrics.
		Str("status", resp.Status).
		Int("body_bytes", respTracker.BytesRead()).
		Int("bytes_read_bps", respTracker.BytesReadPerSecond()).
		Dur("body_read_ms", respTracker.Duration())
}

func generateID() string {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}

	return hex.EncodeToString(bytes)
}

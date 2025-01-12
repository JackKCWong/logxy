package internal

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var _ http.Handler = (*SingleHostProxy)(nil)

type SingleHostProxy struct {
	Target *url.URL
}

// ServeHTTP implements http.Handler.
func (h *SingleHostProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	start := time.Now()
	logevt := log.Info().Str("req_id", generateReqID()).Time("req_arrived_at", start)
	defer logevt.Send()

	reqTracker := &ReaderTracker{Reader: req.Body}
	req.URL.Scheme = h.Target.Scheme
	req.URL.Host = h.Target.Host
	req.Body = reqTracker
	req.RequestURI = ""

	resp, err := http.DefaultClient.Do(req)

	logevt.
		TimeDiff("total_ms", time.Now(), start).
		Dict("req_metrics", zerolog.Dict().
			Str("url", req.Method+" "+req.URL.String()+" "+req.Proto).
			Int("body_bytes", reqTracker.BytesRead()).
			Int("body_read_bps", reqTracker.BytesReadPerSecond()).
			Dur("body_read_ms", reqTracker.Duration()))

	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		logevt.Err(err)
		return
	}

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

	logevt.Dict("resp_metrics", zerolog.Dict().
		Str("status", resp.Status).
		Int("body_bytes", respTracker.BytesRead()).
		Int("bytes_read_bps", respTracker.BytesReadPerSecond()).
		Dur("body_read_ms", respTracker.Duration()))
}

func generateReqID() string {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}

	return hex.EncodeToString(bytes)
}

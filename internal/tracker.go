package internal

import (
	"io"
	"time"
)

var _ io.ReadCloser = (*ReaderTracker)(nil)

type ReaderTracker struct {
	Reader    io.ReadCloser
	ID        string
	bytesRead int
	duration  time.Duration
}

// Close implements io.ReadCloser.
func (r *ReaderTracker) Close() error {
	return r.Reader.Close()
}

func (r *ReaderTracker) Read(p []byte) (n int, err error) {
	start := time.Now()
	n, err = r.Reader.Read(p)
	if n > 0 {
		elapsed := time.Since(start)
		r.duration += elapsed
		r.bytesRead += n
	}

	return n, err
}

func (r *ReaderTracker) BytesRead() int {
	return r.bytesRead
}

func (r *ReaderTracker) Duration() time.Duration {
	return r.duration
}

func (r *ReaderTracker) BytesReadPerSecond() int {
	if r.duration == 0 {
		return 0
	}

	if r.duration < 1*time.Second {
		return r.bytesRead
	}

	return int(float64(r.bytesRead) / r.duration.Seconds())
}

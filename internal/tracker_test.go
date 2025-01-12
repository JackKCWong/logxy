package internal

import (
	"bytes"
	"io"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestReaderTracker(t *testing.T) {
	Convey("BytesReadPerSecond", t, func() {
		Convey("smoke test", func() {
			r := &ReaderTracker{
				bytesRead: 100,
				duration:  time.Second * 5,
			}

			So(r.BytesReadPerSecond(), ShouldEqual, 20)
		})
		Convey("zero duration", func() {
			r := &ReaderTracker{}
			So(r.BytesReadPerSecond(), ShouldEqual, 0)
		})
		Convey("it should be able to read and measure duration", func() {
			r := &ReaderTracker{
				Reader: io.NopCloser(bytes.NewBufferString("hello world")),
			}
			buf := make([]byte, 8)
			n, err := r.Read(buf)
			So(err, ShouldBeNil)
			So(r.BytesRead(), ShouldEqual, n)
			So(r.BytesReadPerSecond(), ShouldBeGreaterThan, 0)
			So(buf, ShouldResemble, []byte("hello wo"))

			_, err = r.Read(buf)
			So(err, ShouldBeNil)
			So(r.BytesRead(), ShouldEqual, 11)
			So(r.BytesReadPerSecond(), ShouldBeGreaterThan, 0)
			So(buf, ShouldResemble, []byte("rldlo wo"))
		})
	})
}

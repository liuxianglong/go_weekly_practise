package accesslog

import (
	"bufio"
	"net"
	"net/http"
	"time"
)

const (
	noWritten     = -1
	defaultStatus = http.StatusOK
)

type ResponseWriter interface {
	http.ResponseWriter
	http.Hijacker
	http.Flusher
	http.CloseNotifier

	// Returns the HTTP response status code of the current request.
	Status() int

	// Returns the number of bytes already written into the response http body.
	// See Written()
	Size() int

	// Time to First Byte
	FirstByteTime() time.Time
}

func NewResponseWriter(w http.ResponseWriter) ResponseWriter {
	writer := &responseWriter{ResponseWriter: w}
	writer.reset(w)
	return writer
}

type responseWriter struct {
	http.ResponseWriter
	size   int
	status int
	fbt    time.Time
}

var _ ResponseWriter = &responseWriter{}

func (w *responseWriter) reset(writer http.ResponseWriter) {
	w.ResponseWriter = writer
	w.size = noWritten
	w.status = defaultStatus
}

func (w *responseWriter) WriteHeader(code int) {
	if code > 0 && w.status != code {
		w.ResponseWriter.WriteHeader(code)
		if w.fbt.IsZero() {
			w.status = code
			w.fbt = time.Now()
		}
	}
}

func (w *responseWriter) Write(data []byte) (n int, err error) {
	if w.fbt.IsZero() {
		w.WriteHeader(http.StatusOK)
	}
	n, err = w.ResponseWriter.Write(data)
	w.size += n
	return
}

func (w *responseWriter) Status() int {
	return w.status
}

func (w *responseWriter) Size() int {
	if w.size == noWritten {
		return 0
	}

	return w.size
}

func (w *responseWriter) FirstByteTime() time.Time {
	return w.fbt
}

// Implements the http.Hijacker interface
func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w.size < 0 {
		w.size = 0
	}
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

// Implements the http.CloseNotify interface
func (w *responseWriter) CloseNotify() <-chan bool {
	return w.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

// Implements the http.Flush interface
func (w *responseWriter) Flush() {
	w.ResponseWriter.(http.Flusher).Flush()
}

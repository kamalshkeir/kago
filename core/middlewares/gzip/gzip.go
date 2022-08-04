package gzip

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"net"
	"net/http"
)

type WrappedResponseWriter struct {
	w       http.ResponseWriter
	gwriter *gzip.Writer
}

func NewWrappedResponseWriter(w http.ResponseWriter) *WrappedResponseWriter {
	gwriter := gzip.NewWriter(w)
	return &WrappedResponseWriter{w, gwriter}
}

func (wrw *WrappedResponseWriter) Header() http.Header {
	return wrw.w.Header()
}

func (wrw *WrappedResponseWriter) WriteHeader(statuscode int) {
	wrw.w.WriteHeader(statuscode)
}

func (wrw *WrappedResponseWriter) Write(d []byte) (int, error) {
	return wrw.gwriter.Write(d)
}

func (wrw *WrappedResponseWriter) Flush() {
	wrw.gwriter.Flush()
	wrw.gwriter.Close()
}

func (wrw *WrappedResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := wrw.w.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("http.Hijacker interface is not supported")
}

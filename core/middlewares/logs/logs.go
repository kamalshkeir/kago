package logs

import (
	"net/http"
)

type StatusRecorder struct {
	http.ResponseWriter
	Status int
}

func (r *StatusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}


/* func (r *StatusRecorder) Flush()  {
	if v,ok := r.ResponseWriter.(http.Flusher);ok {
		v.Flush()
	}
}

func (r *StatusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
    if hj, ok := r.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("LOGS MIDDLEWARE: http.Hijacker interface is not supported")
} */


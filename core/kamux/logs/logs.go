package logs

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils"
	"github.com/kamalshkeir/kago/core/utils/eventbus"
	"github.com/kamalshkeir/kago/core/utils/logger"
)

type StatusRecorder struct {
	http.ResponseWriter
	Status int
}


var LOGS = func(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if utils.StringContains(r.URL.Path, "metrics", "sw.js", "favicon", "/static/", "/sse/", "/ws/", "/wss/") {
			h.ServeHTTP(w, r)
			return
		}
		//check if connection is ws
		for _, header := range r.Header["Upgrade"] {
			if header == "websocket" {
				// connection is ws
				h.ServeHTTP(w, r)
				return
			}
		}
		recorder := &StatusRecorder{
			ResponseWriter: w,
			Status:         200,
		}
		t := time.Now()
		h.ServeHTTP(recorder, r)
		res := fmt.Sprintf("[%s] --> '%s' --> [%d]  from: %s ---------- Took: %v", r.Method, r.URL.Path, recorder.Status, r.RemoteAddr, time.Since(t))

		if recorder.Status >= 200 && recorder.Status < 400 {
			fmt.Printf(logger.Green, res)
		} else if recorder.Status >= 400 || recorder.Status < 200 {
			fmt.Printf(logger.Red, res)
		} else {
			fmt.Printf(logger.Yellow, res)
		}
		if settings.Config.Logs {
			logger.StreamLogs = append(logger.StreamLogs, res)
			eventbus.Publish("internal-logs", map[string]string{})
		}
	})
}


func (r *StatusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *StatusRecorder) Flush()  {
	if v,ok := r.ResponseWriter.(http.Flusher);ok {
		v.Flush()
	}
}

func (r *StatusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
    if hj, ok := r.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("LOGS MIDDLEWARE: http.Hijacker interface is not supported")
}

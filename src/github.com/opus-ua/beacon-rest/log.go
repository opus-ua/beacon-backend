package beaconrest

import (
    "net/http"
    "time"
    "strings"
    "log"
    "fmt"
    "strconv"
)

/*
    Credit for this gist goes to cespare (github)
*/

type ApacheLogRecord struct {
    http.ResponseWriter

    ip string
    time    time.Time
    method, uri, protocol string
    status int
    responseBytes int64
    elapsedTime time.Duration
}

func (r *ApacheLogRecord) Log() {
    requestLine := fmt.Sprintf("%s %s %s", r.method, r.uri, r.protocol)
    timeInt := r.elapsedTime.Nanoseconds() / int64(time.Millisecond)
    timeStr := fmt.Sprintf("%dms", strconv.FormatInt(timeInt, 10))
    if timeInt < 10 {
        timeInt = r.elapsedTime.Nanoseconds() / int64(time.Microsecond)
        timeStr = fmt.Sprintf("%sμs", strconv.FormatInt(timeInt, 10))
    }
    ApacheFormatPattern := "%s \"%s %d %d bytes\" %s\n"
    log.Printf(ApacheFormatPattern, r.ip, requestLine,
                r.status, r.responseBytes, timeStr)
}

func (r *ApacheLogRecord) Write(p []byte) (int, error) {
    written, err := r.ResponseWriter.Write(p)
    r.responseBytes += int64(written)
    return written, err
}

func (r *ApacheLogRecord) WriteHeader(status int) {
    r.status = status
    r.ResponseWriter.WriteHeader(status)
}

type ApacheLoggingHandler struct {
    handler http.Handler
}

func NewApacheLoggingHandler(handler http.Handler) http.Handler {
   return &ApacheLoggingHandler{
        handler: handler,
   }
}

func (h *ApacheLoggingHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
    clientIP := r.RemoteAddr
    if colon := strings.LastIndex(clientIP, ":"); colon != -1 {
        clientIP = clientIP[:colon]
    }

    record := &ApacheLogRecord{
        ResponseWriter: rw,
        ip: clientIP,
        time: time.Time{},
        method: r.Method,
        uri: r.RequestURI,
        protocol: r.Proto,
        status: http.StatusOK,
        elapsedTime: time.Duration(0),
    }

    startTime := time.Now()
    h.handler.ServeHTTP(record, r)
    finishTime := time.Now()

    record.time = finishTime.UTC()
    record.elapsedTime = finishTime.Sub(startTime)

    record.Log()
}

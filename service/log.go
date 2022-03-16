package service

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

// LoggerDefault return default general purpose
// logger that can be used everywhere across project.
//
// It should be initialized once and then reused or modified
// in different areas.
func LoggerDefault() *logrus.Logger {
	return logrus.New()
}

// LoggerLogFormatter is adapter which implements chi LogFormatter
// interface for logrus Logger.
type LoggerLogFormatter struct {
	*logrus.Logger
}

// NewLogEntry returns local LogEntry instance for the scope of given
// request.
func (log *LoggerLogFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	return loggerLogEntry{
		entry: log.WithTime(time.Now()).WithFields(logrus.Fields{
			"method": r.Method,
			"from":   r.RemoteAddr,
			"reqID":  middleware.GetReqID(r.Context()),
		}),
		req: r,
	}
}

// loggerLogEntry is adapter which implements chi LogEntry
// interface for logrus Logger.
type loggerLogEntry struct {
	entry *logrus.Entry
	req   *http.Request
}

func (la loggerLogEntry) Write(
	status, bytes int,
	header http.Header,
	elapsed time.Duration,
	extra interface{},
) {
	scheme := "http"
	if la.req.TLS != nil {
		scheme = "https"
	}

	la.entry.WithFields(logrus.Fields{
		"status":  status,
		"bytes":   bytes,
		"elapsed": elapsed.String(),
	}).Infof("%s %s://%s%s %s\" ", la.req.Method, scheme, la.req.Host, la.req.RequestURI, la.req.Proto)
}

func (log loggerLogEntry) Panic(v interface{}, stack []byte) {
	middleware.PrintPrettyStack(v)
}

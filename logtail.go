// Copyright 2016 Casey Marshall.

// Package logtail provides an HTTP handler that serves log file contents.
package logtail

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
)

const (
	defaultOffset = -2048
	defaultLimit  = 2048
)

// LogTail is an HTTP handler that serves a log file.
type LogTail struct {
	logger ErrorLogger
	path   string
	redact *regexp.Regexp
}

// NewLogTail returns a new LogTail instance.
func NewLogTail(path string, redact *regexp.Regexp, logger ErrorLogger) *LogTail {
	if logger == nil {
		logger = &defaultLogger{}
	}
	return &LogTail{
		logger: logger,
		path:   path,
		redact: redact,
	}
}

// ServeHTTP implements http.Handler.
func (t *LogTail) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params, err := newRequestParams(r.URL.Query())
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	path := t.path
	if params.suffix != "" {
		path = path + "." + params.suffix
	}
	// TODO: automatically detect & uncompress rotated log files.
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		t.logger.Errorf("failed to open %q: %v", path, err)
		return
	}
	defer f.Close()

	fInfo, err := f.Stat()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		t.logger.Errorf("failed to stat file: %v", err)
		return
	}
	w.Header().Set("LogTail-File-Length", fmt.Sprintf("%d", fInfo.Size()))

	if 0-params.offset >= fInfo.Size() {
		params.offset = 0
		params.limit = int(fInfo.Size())
	}
	if params.offset >= fInfo.Size() {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var whence int
	if params.offset < 0 {
		whence = 2
	}
	_, err = f.Seek(params.offset, whence)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		t.logger.Errorf("failed to seek to offset %d: %v", params.offset, err)
		return
	}

	var contents io.Reader = f
	if t.redact != nil {
		buf := make([]byte, params.limit)
		_, err := f.Read(buf)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			t.logger.Errorf("failed to read contents: %v", err)
		}
		buf = t.redact.ReplaceAllFunc(buf, redact)
		contents = bytes.NewBuffer(buf)
	}

	w.WriteHeader(http.StatusOK)
	_, err = io.CopyN(w, contents, int64(params.limit))
	if err != nil {
		t.logger.Errorf("error writing response: %v", err)
	}
}

func redact(b []byte) []byte {
	return bytes.Repeat([]byte("█"), len(b)) // Unicode FULL BLOCK U+2588
}

type requestParams struct {
	offset int64
	limit  int
	suffix string
}

func newRequestParams(v url.Values) (*requestParams, error) {
	var (
		result requestParams
		err    error
	)
	if offset := v.Get("offset"); offset != "" {
		result.offset, err = strconv.ParseInt(offset, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid offset %q: %v", offset, err)
		}
	} else {
		result.offset = defaultOffset
	}

	if limit := v.Get("limit"); limit != "" {
		result.limit, err = strconv.Atoi(limit)
		if err != nil {
			return nil, fmt.Errorf("invalid limit %q: %v", limit, err)
		}
	} else {
		result.limit = defaultLimit
	}

	if suffix := v.Get("suffix"); suffix != "" {
		suffixInt, err := strconv.Atoi(suffix)
		if err != nil {
			return nil, fmt.Errorf("invalid suffix %q: %v", suffix, err)
		}
		if suffixInt < 0 || suffixInt > 10 {
			return nil, fmt.Errorf("invalid suffix %q", suffix)
		}
	}
	return &result, nil
}

// ErrorLogger is an interface used for logging errors.
type ErrorLogger interface {
	Errorf(format string, args ...interface{})
}

type defaultLogger struct{}

// Errorf implements ErrorLogger.
func (*defaultLogger) Errorf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

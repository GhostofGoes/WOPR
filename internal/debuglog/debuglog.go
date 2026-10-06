// Package debuglog is wopr's opt-in debug log (docs/PLAN.md §8): with WOPR_DEBUG set, it
// writes to os.UserCacheDir()/wopr/debug.log, a 0600 file in a 0700 directory. It records
// the session seed, the terminal and slow work; it never records what the player types.
package debuglog

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// Log is an open debug log. A nil *Log is valid and discards everything, so callers need
// no checks.
type Log struct {
	l    *log.Logger
	c    io.Closer
	path string
}

// Open opens the log in dir (normally os.UserCacheDir()), appending.
func Open(dir string) (*Log, error) {
	d := filepath.Join(dir, "wopr")
	if err := os.MkdirAll(d, 0o700); err != nil {
		return nil, fmt.Errorf("debug log: %w", err)
	}
	path := filepath.Join(d, "debug.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("debug log: %w", err)
	}
	return &Log{l: log.New(f, "", log.LstdFlags|log.Lmicroseconds), c: f, path: path}, nil
}

// Printf writes one line. It is safe to call from any goroutine.
func (l *Log) Printf(format string, args ...any) {
	if l == nil {
		return
	}
	l.l.Printf(format, args...)
}

// Path is the log file's path, or "" for a nil log.
func (l *Log) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

// Close closes the log.
func (l *Log) Close() error {
	if l == nil {
		return nil
	}
	return l.c.Close()
}

package logx

import (
	"encoding/json"
	"io"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

type Level int8

const (
	LevelInfo Level = iota
	LevelError
	LevelFatal
	LevelOff
)

func (l Level) String() string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return ""
	}
}

type Logger struct {
	out   io.Writer
	level Level
	mu    sync.Mutex
}

func NewLogger(out io.Writer, level Level) *Logger {
	return &Logger{
		out:   out,
		level: level,
	}
}

func (l *Logger) Info(message string, properties map[string]string) {
	l.log(LevelInfo, message, properties)
}

func (l *Logger) Error(err error, properties map[string]string) {
	l.log(LevelError, err.Error(), properties)
}

func (l *Logger) Fatal(err error, properties map[string]string) {
	l.log(LevelFatal, err.Error(), properties)
	os.Exit(1)
}

func (l *Logger) log(level Level, message string, properties map[string]string) (int, error) {
	if level < l.level {
		return 0, nil
	}

	aux := struct {
		Level      string            `json:"level"`
		Time       string            `json:"time"`
		Message    string            `json:"message"`
		Properties map[string]string `json:"properties,omitempty"`
		Trace      string            `json:"trace,omitempty"`
	}{
		Level:      level.String(),
		Time:       time.Now().UTC().Format(time.RFC3339),
		Message:    message,
		Properties: properties,
	}

	if level >= LevelError {
		aux.Trace = string(debug.Stack())
	}

	var line []byte
	line, err := json.Marshal(aux)
	if err != nil {
		line = []byte(LevelError.String() + ": unable to marshal log message" + err.Error())
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	return l.out.Write(append(line, '\n'))
}

// Write implements io.Writer.
func (l *Logger) Write(message []byte) (n int, err error) {
	return l.log(LevelError, string(message), nil)
}

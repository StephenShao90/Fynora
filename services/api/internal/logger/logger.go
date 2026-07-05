package logger

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"time"
)

type Logger struct{ base *log.Logger }

func New() Logger {
	return Logger{base: log.New(os.Stdout, "", 0)}
}

func (l Logger) Info(message string, fields map[string]interface{}) {
	l.write("info", message, fields)
}

func (l Logger) Error(message string, fields map[string]interface{}) {
	l.write("error", message, fields)
}

func (l Logger) write(level, message string, fields map[string]interface{}) {
	if l.base == nil {
		l.base = log.New(io.Discard, "", 0)
	}
	if fields == nil {
		fields = map[string]interface{}{}
	}
	fields["level"] = level
	fields["message"] = message
	fields["ts"] = time.Now().UTC().Format(time.RFC3339Nano)
	b, _ := json.Marshal(fields)
	l.base.Println(string(b))
}

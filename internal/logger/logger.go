package logger

import (
	"log"
	"os"
)

var l *log.Logger

func Init(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	l = log.New(f, "", log.Ltime|log.Lshortfile)
	return nil
}

func Log(format string, args ...any) {
	if l != nil {
		l.Printf(format, args...)
	}
}

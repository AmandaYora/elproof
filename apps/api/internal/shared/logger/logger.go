// Package logger provides a minimal, dependency-free request/error logger.
package logger

import (
	"log"
	"os"
)

var std = log.New(os.Stdout, "", log.LstdFlags)

func Info(format string, args ...interface{}) {
	std.Printf("[INFO] "+format, args...)
}

func Error(format string, args ...interface{}) {
	std.Printf("[ERROR] "+format, args...)
}

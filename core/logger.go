package core

import (
	"fmt"
	"gin-core/core/color"
	"io"
	"log"
)

type Logger interface {
	Trace(v ...any)
	Debug(v ...any)
	Info(v ...any)
	Warn(v ...any)
	Error(v ...any)
}

type sampleLogger struct {
	*log.Logger
}

// Println 日志输出
func (s *sampleLogger) Println(a ...any) {
	_ = s.Output(3, fmt.Sprintln(a...))
}

type logger struct {
	trace *sampleLogger
	debug *sampleLogger
	info  *sampleLogger
	warn  *sampleLogger
	error *sampleLogger
}

func (l *logger) Trace(v ...any) {
	l.trace.Println(v...)
}

func (l *logger) Debug(v ...any) {
	l.debug.Println(v...)
}

func (l *logger) Info(v ...any) {
	l.info.Println(v...)
}

func (l *logger) Warn(v ...any) {
	l.warn.Println(v...)
}

func (l *logger) Error(v ...any) {
	l.error.Println(v...)
}

func newLogger(writer io.Writer, prefix string) *sampleLogger {
	return &sampleLogger{
		log.New(
			writer,
			fmt.Sprintf("[%-14s]  ", prefix),
			log.Ldate|log.Ltime|log.Llongfile),
	}
}

func newStdLogger(prefix string) *sampleLogger {
	return newLogger(log.Writer(), prefix)
}

func NewLogger() Logger {
	return &logger{
		trace: newStdLogger(color.FormatColor(97, "TRACE")),
		debug: newStdLogger(color.FormatColor(91, "DEBUG")),
		info:  newStdLogger(color.FormatColor(92, "INFO")),
		warn:  newStdLogger(color.FormatColor(93, "WARN")),
		error: newStdLogger(color.FormatColor(91, "ERROR")),
	}
}

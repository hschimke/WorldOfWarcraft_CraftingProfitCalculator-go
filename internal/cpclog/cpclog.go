package cpclog

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

const (
	SILLY uint = iota
	DEBUG
	INFO
	ERROR
)

const (
	LevelSilly slog.Level = slog.LevelDebug - 1
)

type CpCLog struct {
	LogLevel uint
	logger   *slog.Logger
}

func NewCpCLog(level uint) *CpCLog {
	var slogLevel slog.Level
	switch level {
	case SILLY:
		slogLevel = LevelSilly
	case DEBUG:
		slogLevel = slog.LevelDebug
	case INFO:
		slogLevel = slog.LevelInfo
	case ERROR:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}
	handler := slog.NewTextHandler(os.Stderr, opts)
	return &CpCLog{
		LogLevel: level,
		logger:   slog.New(handler),
	}
}

func GetLevel(level string) (ret_level uint) {
	switch strings.ToLower(level) {
	case "silly":
		ret_level = SILLY
	case "debug":
		ret_level = DEBUG
	case "info":
		ret_level = INFO
	case "error":
		ret_level = ERROR
	default:
		ret_level = INFO
	}
	return
}

func (l *CpCLog) log(level slog.Level, msg string, args ...any) {
	if l.logger == nil {
		l.logger = slog.Default()
	}
	l.logger.Log(context.Background(), level, msg, args...)
}

func (l *CpCLog) Silly(v ...any) {
	l.log(LevelSilly, fmt.Sprint(v...))
}

func (l *CpCLog) Debug(v ...any) {
	l.log(slog.LevelDebug, fmt.Sprint(v...))
}

func (l *CpCLog) Info(v ...any) {
	l.log(slog.LevelInfo, fmt.Sprint(v...))
}

func (l *CpCLog) Error(v ...any) {
	l.log(slog.LevelError, fmt.Sprint(v...))
}

func (l *CpCLog) Fatal(v ...any) {
	l.Error(v...)
	os.Exit(1)
}

func (l *CpCLog) Sillyf(format string, v ...any) {
	l.log(LevelSilly, fmt.Sprintf(format, v...))
}

func (l *CpCLog) Debugf(format string, v ...any) {
	l.log(slog.LevelDebug, fmt.Sprintf(format, v...))
}

func (l *CpCLog) Infof(format string, v ...any) {
	l.log(slog.LevelInfo, fmt.Sprintf(format, v...))
}

func (l *CpCLog) Errorf(format string, v ...any) {
	l.log(slog.LevelError, fmt.Sprintf(format, v...))
}

func (l *CpCLog) Fatalf(format string, v ...any) {
	l.Errorf(format, v...)
	os.Exit(1)
}

package cpclog

import (
	"log"
	"strings"
)

const (
	SILLY uint = iota
	DEBUG
	INFO
	ERROR
)

type CpCLog struct {
	LogLevel uint
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

func (l *CpCLog) Silly(v ...any) {
	if l.LogLevel <= SILLY {
		log.Print(v...)
	}
}

func (l *CpCLog) Debug(v ...any) {
	if l.LogLevel <= DEBUG {
		log.Print(v...)
	}
}

func (l *CpCLog) Info(v ...any) {
	if l.LogLevel <= INFO {
		log.Print(v...)
	}
}

func (l *CpCLog) Error(v ...any) {
	if l.LogLevel <= ERROR {
		log.Print(v...)
	}
}

func (l *CpCLog) Fatal(v ...any) {
	log.Fatal(v...)
}

func (l *CpCLog) Sillyf(format string, v ...any) {
	if l.LogLevel <= SILLY {
		log.Printf(format, v...)
	}
}

func (l *CpCLog) Debugf(format string, v ...any) {
	if l.LogLevel <= DEBUG {
		log.Printf(format, v...)
	}
}

func (l *CpCLog) Infof(format string, v ...any) {
	if l.LogLevel <= INFO {
		log.Printf(format, v...)
	}
}

func (l *CpCLog) Errorf(format string, v ...any) {
	if l.LogLevel <= ERROR {
		log.Printf(format, v...)
	}
}

func (l *CpCLog) Fatalf(format string, v ...any) {
	log.Fatalf(format, v...)
}

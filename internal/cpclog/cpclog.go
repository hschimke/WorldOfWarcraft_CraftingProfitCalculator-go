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

var LogLevel uint = INFO

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

func Silly(v ...interface{}) {
	if LogLevel <= SILLY {
		log.Print(v...)
	}
}

func Debug(v ...interface{}) {
	if LogLevel <= DEBUG {
		log.Print(v...)
	}
}

func Info(v ...interface{}) {
	if LogLevel <= INFO {
		log.Print(v...)
	}
}

func Error(v ...interface{}) {
	if LogLevel <= ERROR {
		log.Print(v...)
	}
}

func Fatal(v ...interface{}) {
	log.Fatal(v...)
}

func Sillyf(format string, v ...interface{}) {
	if LogLevel <= SILLY {
		log.Printf(format, v...)
	}
}

func Debugf(format string, v ...interface{}) {
	if LogLevel <= DEBUG {
		log.Printf(format, v...)
	}
}

func Infof(format string, v ...interface{}) {
	if LogLevel <= INFO {
		log.Printf(format, v...)
	}
}

func Errorf(format string, v ...interface{}) {
	if LogLevel <= ERROR {
		log.Printf(format, v...)
	}
}

func Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

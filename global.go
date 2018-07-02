package main

import (
	"fmt"
	"log"
	"os"
)

type ConsoleLog struct {
	*log.Logger
}

const (
	logFlags = log.Ldate | log.Ltime
)

var (
	InfoLog ConsoleLog
	WarnLog ConsoleLog
	ErrLog  ConsoleLog
)

func init() {
	InfoLog = ConsoleLog{Logger: log.New(os.Stdout, "[ ] ", logFlags)}
	WarnLog = ConsoleLog{Logger: log.New(os.Stderr, "[*] ", logFlags)}
	ErrLog = ConsoleLog{Logger: log.New(os.Stderr, "[!] ", logFlags)}
}

func (l *ConsoleLog) Log(v ...interface{}) {
	s := fmt.Sprint(v)
	l.Printf("| %s", s)
}

func (l *ConsoleLog) Logf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v)
	l.Printf("| %s", s)
}

func (l *ConsoleLog) Logln(v ...interface{}) {
	s := fmt.Sprintln(v)
	l.Printf("| %s", s)
}

func (l *ConsoleLog) Die(c int, v ...interface{}) {
	s := fmt.Sprint(v)
	l.Printf("| %s", s)
	os.Exit(c)
}

func (l *ConsoleLog) Dief(c int, format string, v ...interface{}) {
	s := fmt.Sprintf(format, v)
	l.Printf("| %s", s)
	os.Exit(c)
}

func (l *ConsoleLog) Dieln(c int, v ...interface{}) {
	s := fmt.Sprintln(v)
	l.Printf("| %s", s)
	os.Exit(c)
}

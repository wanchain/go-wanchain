// +build windows,plan9

// because syslog is incompatible with windows and plan9, so only print local log

package log

import (
	"fmt"
)

type SyslogFun func(m string) error
type LocallogFun func(msg string, ctx ...interface{})

type Syslogger struct {
}

var (
	syslogger Syslogger
)

func InitSyslog(net, svr, level, tag string) error {
	Info("because syslog is incompatible with windows and plan9, so only print local log")
	return nil
}

func CloseSyslog() {
}

func SyslogDebug(format string, a ...interface{}) {
	logStr := fmt.Sprintf(format, a...)
	Debug(logStr)
}

func SyslogInfo(format string, a ...interface{}) {
	logStr := fmt.Sprintf(format, a...)
	Info(logStr)
}

func SyslogNotice(format string, a ...interface{}) {
	logStr := fmt.Sprintf(format, a...)
	Info(logStr)
}

func SyslogWarning(format string, a ...interface{}) {
	logStr := fmt.Sprintf(format, a...)
	Warn(logStr)
}

func SyslogErr(format string, a ...interface{}) {
	logStr := fmt.Sprintf(format, a...)
	Error(logStr)
}

func SyslogCrit(format string, a ...interface{}) {
	logStr := fmt.Sprintf(format, a...)
	Error(logStr)
}

func SyslogAlert(format string, a ...interface{}) {
	logStr := fmt.Sprintf(format, a...)
	Error(logStr)
}

func SyslogEmerg(format string, a ...interface{}) {
	logStr := fmt.Sprintf(format, a...)
	Error(logStr)
}


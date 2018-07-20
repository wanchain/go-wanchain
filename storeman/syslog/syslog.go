package syslog

import (
	"fmt"
	"github.com/wanchain/go-wanchain/log"
	"log/syslog"
)

var syslogger *syslog.Writer

func StartSyslog(net, svr, level, tag string) error {
	log.Info("mpc syslog config", "net", net, "svr", svr, "level", level, "tag", tag)

	priority := syslog.LOG_INFO
	switch level {
	case "EMERG":
		priority = syslog.LOG_EMERG
	case "ALERT":
		priority = syslog.LOG_ALERT
	case "CRIT":
		priority = syslog.LOG_CRIT
	case "ERR":
		priority = syslog.LOG_ERR
	case "WARNING":
		priority = syslog.LOG_WARNING
	case "NOTICE":
		priority = syslog.LOG_NOTICE
	case "INFO":
		priority = syslog.LOG_INFO
	case "DEBUG":
		priority = syslog.LOG_DEBUG
	}

	var err error
	syslogger, err = syslog.Dial(net, svr, priority, tag)
	if err != nil {
		log.Error("init syslog fail", "err", err)
	}

	return nil
}

func Debug(format string, a ...interface{}) {
	if syslogger == nil {
		return
	}

	syslogger.Debug(fmt.Sprintf(format, a...))
}

func Info(format string, a ...interface{}) {
	if syslogger == nil {
		return
	}

	syslogger.Info(fmt.Sprintf(format, a...))
}

func Notice(format string, a ...interface{}) {
	if syslogger == nil {
		return
	}

	syslogger.Notice(fmt.Sprintf(format, a...))
}

func Warning(format string, a ...interface{}) {
	if syslogger == nil {
		return
	}

	syslogger.Warning(fmt.Sprintf(format, a...))
}

func Err(format string, a ...interface{}) {
	if syslogger == nil {
		return
	}

	syslogger.Err(fmt.Sprintf(format, a...))
}

func Crit(format string, a ...interface{}) {
	if syslogger == nil {
		return
	}

	syslogger.Crit(fmt.Sprintf(format, a...))
}

func Alert(format string, a ...interface{}) {
	if syslogger == nil {
		return
	}

	syslogger.Alert(fmt.Sprintf(format, a...))
}

func Emerg(format string, a ...interface{}) {
	if syslogger == nil {
		return
	}

	syslogger.Emerg(fmt.Sprintf(format, a...))
}

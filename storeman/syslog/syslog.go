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
	logStr := fmt.Sprintf(format, a...)
	log.Debug(logStr)
	if syslogger == nil {
		return
	}

	syslogger.Debug(logStr)
}

func Info(format string, a ...interface{}) {
	logStr := fmt.Sprintf(format, a...)
	log.Info(logStr)
	if syslogger == nil {
		return
	}

	syslogger.Info(logStr)
}

func Notice(format string, a ...interface{}) {
	logStr := fmt.Sprintf(format, a...)
	log.Info(logStr)
	if syslogger == nil {
		return
	}

	syslogger.Notice(logStr)
}

func Warning(format string, a ...interface{}) {
	logStr := fmt.Sprintf(format, a...)
	log.Warn(logStr)
	if syslogger == nil {
		return
	}

	syslogger.Warning(logStr)
}

func Err(format string, a ...interface{}) {
	logStr := fmt.Sprintf(format, a...)
	log.Error(logStr)
	if syslogger == nil {
		return
	}

	syslogger.Err(logStr)
}

func Crit(format string, a ...interface{}) {
	logStr := fmt.Sprintf(format, a...)
	log.Warn(logStr)
	if syslogger == nil {
		return
	}

	syslogger.Crit(logStr)
}

func Alert(format string, a ...interface{}) {
	logStr := fmt.Sprintf(format, a...)
	log.Warn(logStr)
	if syslogger == nil {
		return
	}

	syslogger.Alert(logStr)
}

func Emerg(format string, a ...interface{}) {
	logStr := fmt.Sprintf(format, a...)
	log.Warn(logStr)
	if syslogger == nil {
		return
	}

	syslogger.Emerg(logStr)
}

// +build !windows,!plan9

package log

import (
	"errors"
	"fmt"
	"log/syslog"
)

type SyslogFun func(m string) error
type LocallogFun func(msg string, ctx ...interface{})

type SyslogSetting struct {
	net string
	svr string
	level string
	tag string
}

type Syslogger struct {
	writer *syslog.Writer
	threshold syslog.Priority
	setting *SyslogSetting
	dialTimes uint64
}

var (
	syslogger Syslogger
)

func InitSyslog(net, svr, level, tag string) error {
	Info("mpc syslog config", "net", net, "svr", svr, "level", level, "tag", tag)
	if syslogger.writer != nil {
		err := errors.New("repetitive initialization syslog")
		Error(err.Error())
		return err
	}

	syslogger.threshold = syslog.LOG_INFO
	switch level {
	case "EMERG":
		syslogger.threshold = syslog.LOG_EMERG
	case "ALERT":
		syslogger.threshold = syslog.LOG_ALERT
	case "CRIT":
		syslogger.threshold = syslog.LOG_CRIT
	case "ERROR":
		syslogger.threshold = syslog.LOG_ERR
	case "WARNING":
		syslogger.threshold = syslog.LOG_WARNING
	case "NOTICE":
		syslogger.threshold = syslog.LOG_NOTICE
	case "INFO":
		syslogger.threshold = syslog.LOG_INFO
	case "DEBUG":
		syslogger.threshold = syslog.LOG_DEBUG
	}

	syslogger.setting = &SyslogSetting {
		net,
		svr,
		level,
		tag,
	}

	return dialSyslog()
}

func dialSyslog() error {
	syslogger.dialTimes++
	if syslogger.dialTimes%10 != 1 {
		err := errors.New("delay retry connect.")
		Error("dial syslog fail", "err", err, "retry times", syslogger.dialTimes)
		return err
	}

	var err error
	syslogger.writer, err = syslog.Dial(syslogger.setting.net, syslogger.setting.svr, syslog.LOG_INFO, syslogger.setting.tag)
	if err != nil {
		Error("dial syslog fail", "err", err, "retry times", syslogger.dialTimes)
		return err
	}

	return nil
}

func CloseSyslog() {
	if syslogger.writer == nil {
		return
	}

	syslogger.writer.Close()
	syslogger.writer = nil
}

func SyslogDebug(format string, a ...interface{}) {
	writeSyslog(syslog.LOG_DEBUG, format, a...)
}

func SyslogInfo(format string, a ...interface{}) {
	writeSyslog(syslog.LOG_INFO, format, a...)
}

func SyslogNotice(format string, a ...interface{}) {
	writeSyslog(syslog.LOG_NOTICE, format, a...)
}

func SyslogWarning(format string, a ...interface{}) {
	writeSyslog(syslog.LOG_WARNING, format, a...)
}

func SyslogErr(format string, a ...interface{}) {
	writeSyslog(syslog.LOG_ERR, format, a...)
}

func SyslogCrit(format string, a ...interface{}) {
	writeSyslog(syslog.LOG_CRIT, format, a...)
}

func SyslogAlert(format string, a ...interface{}) {
	writeSyslog(syslog.LOG_ALERT, format, a...)
}

func SyslogEmerg(format string, a ...interface{}) {
	writeSyslog(syslog.LOG_EMERG, format, a...)
}


func writeSyslog(level syslog.Priority, format string, a ...interface{}) {
	var sfunc SyslogFun
	var lfunc LocallogFun

	if syslogger.setting != nil && syslogger.writer == nil {
		dialSyslog()
	}

	switch level {
	case syslog.LOG_DEBUG:
		if syslogger.writer != nil {
			sfunc = syslogger.writer.Debug
		}
		lfunc = Debug
	case syslog.LOG_INFO:
		if syslogger.writer != nil {
			sfunc = syslogger.writer.Info
		}
		lfunc = Info
	case syslog.LOG_NOTICE:
		if syslogger.writer != nil {
			sfunc = syslogger.writer.Notice
		}
		lfunc = Info
	case syslog.LOG_WARNING:
		if syslogger.writer != nil {
			sfunc = syslogger.writer.Warning
		}
		lfunc = Warn
	case syslog.LOG_ERR:
		if syslogger.writer != nil {
			sfunc = syslogger.writer.Err
		}
		lfunc = Error
	case syslog.LOG_CRIT:
		if syslogger.writer != nil {
			sfunc = syslogger.writer.Crit
		}
		lfunc = Error
	case syslog.LOG_ALERT:
		if syslogger.writer != nil {
			sfunc = syslogger.writer.Alert
		}
		lfunc = Error
	case syslog.LOG_EMERG:
		if syslogger.writer != nil {
			sfunc = syslogger.writer.Emerg
		}
		lfunc = Error
	}

	logStr := fmt.Sprintf(format, a...)
	lfunc(logStr)

	if level <= syslogger.threshold && sfunc != nil {
		if err := sfunc(logStr); err != nil {
			Error("send syslog fail", "err", err)
		}
	}
}


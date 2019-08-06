// +build !windows,!plan9

package log

import (
	"errors"
	"fmt"
	"github.com/wanchain/go-wanchain/event"
	"log/syslog"
)

type SyslogFun func(m string) error
type LocallogFun func(msg string, ctx ...interface{})

const (
	// Severity.
	// From /usr/include/sys/syslog.h.
	// These are the same on Linux, BSD, and OS X.
	LOG_EMERG = iota
	LOG_ALERT
	LOG_CRIT
	LOG_ERR
	LOG_WARNING
	LOG_NOTICE
	LOG_INFO
	LOG_DEBUG
)

type LogInfo struct {
	Lvl syslog.Priority	`json:"level"`
	Msg string			`json:"msg"`
}

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

	alarmFeed    event.Feed
	scope        event.SubscriptionScope

	warnCount	uint64
	wrongCount	uint64
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
	// close all event subscribe
	syslogger.scope.Close()
	if syslogger.writer == nil {
		return
	}

	syslogger.writer.Close()
	syslogger.writer = nil
}

func SyslogDebug(a ...interface{}) {
	writeSyslog(syslog.LOG_DEBUG, a...)
}

func SyslogInfo(a ...interface{}) {
	writeSyslog(syslog.LOG_INFO, a...)
}

func SyslogNotice(a ...interface{}) {
	writeSyslog(syslog.LOG_NOTICE, a...)
}

func SyslogWarning(a ...interface{}) {
	syslogger.warnCount++
	writeSyslog(syslog.LOG_WARNING, a...)
}

func SyslogErr(a ...interface{}) {
	syslogger.wrongCount++
	writeSyslog(syslog.LOG_ERR, a...)
}

func SyslogCrit(a ...interface{}) {
	syslogger.wrongCount++
	writeSyslog(syslog.LOG_CRIT, a...)
}

func SyslogAlert(a ...interface{}) {
	syslogger.wrongCount++
	writeSyslog(syslog.LOG_ALERT, a...)
}

func SyslogEmerg(a ...interface{}) {
	syslogger.wrongCount++
	writeSyslog(syslog.LOG_EMERG, a...)
}

func GetWarnAndWrongLogCount() (uint64, uint64) {
	return syslogger.warnCount, syslogger.wrongCount
}


func writeSyslog(level syslog.Priority, a ...interface{}) {
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

	p := make([]interface{}, 0, len(a)*2)
	for i := range a {
		if i == 0 {
			fastr, ok := a[i].(string)
			if !ok {
				fmt.Println("invalid syslog first input param, need string")
				return
			}

			minLen := 41
			if len(fastr) >= minLen {
				minLen += 10
			}

			fmtstr := fmt.Sprintf("%%-%ds", minLen)
			p = append(p, fmt.Sprintf(fmtstr, a[i]))
		} else if i%2 == 1 {
			p = append(p, a[i])
			p = append(p, string("="))
		} else {
			p = append(p, a[i])
			p = append(p, string(" "))
		}
	}

	logStr := fmt.Sprint(p...)
	if level <= syslog.LOG_CRIT {
		go syslogger.alarmFeed.Send(LogInfo{level, logStr})
	}

	lfunc(logStr)
	if level <= syslogger.threshold && sfunc != nil {
		if err := sfunc(logStr); err != nil {
			Error("send syslog fail", "err", err)
		}
	}
}


func SubscribeAlarm(ch chan<- LogInfo) event.Subscription {
	return syslogger.scope.Track(syslogger.alarmFeed.Subscribe(ch))
}
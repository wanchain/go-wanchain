// because syslog is incompatible with windows and plan9, so only print local log

package log

import (
	"fmt"
	"github.com/wanchain/go-wanchain/event"
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
	Lvl uint32	`json:"level"`
	Msg string			`json:"msg"`
}

type Syslogger struct {
	scope        event.SubscriptionScope
}

var (
	syslogger Syslogger
)

func InitSyslog(net, svr, level, tag string) error {
	Info("because syslog is incompatible with windows and plan9, so only print local log")
	return nil
}

func CloseSyslog() {
	syslogger.scope.Close()
}

func SyslogDebug(a ...interface{}) {
	writeSyslog(LOG_DEBUG, a...)
}

func SyslogInfo(a ...interface{}) {
	writeSyslog(LOG_INFO, a...)
}

func SyslogNotice(a ...interface{}) {
	writeSyslog(LOG_NOTICE, a...)
}

func SyslogWarning(a ...interface{}) {
	writeSyslog(LOG_WARNING, a...)
}

func SyslogErr(a ...interface{}) {
	writeSyslog(LOG_ERR, a...)
}

func SyslogCrit(a ...interface{}) {
	writeSyslog(LOG_CRIT, a...)
}

func SyslogAlert(a ...interface{}) {
	writeSyslog(LOG_ALERT, a...)
}

func SyslogEmerg(a ...interface{}) {
	writeSyslog(LOG_EMERG, a...)
}

func writeSyslog(level uint32, a ...interface{}) {
	var lfunc LocallogFun

	switch level {
	case LOG_DEBUG:
		lfunc = Debug
	case LOG_INFO:
		lfunc = Info
	case LOG_NOTICE:
		lfunc = Info
	case LOG_WARNING:
		lfunc = Warn
	case LOG_ERR:
		lfunc = Error
	case LOG_CRIT:
		lfunc = Error
	case LOG_ALERT:
		lfunc = Error
	case LOG_EMERG:
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
	lfunc(logStr)
}

func SubscribeAlarm(ch chan<- LogInfo) event.Subscription {
	return syslogger.scope.Track(new(event.Feed).Subscribe(ch))
}


func GetWarnAndWrongLogCount() (uint64, uint64) {
	return 0, 0
}

package log

import (
	"fmt"
	"log/syslog"
	"os"
)

var Facilities map[string]syslog.Priority

func init() {
	Facilities = map[string]syslog.Priority{
		"LOG_KERN":     syslog.LOG_KERN,
		"LOG_USER":     syslog.LOG_USER,
		"LOG_MAIL":     syslog.LOG_MAIL,
		"LOG_DAEMON":   syslog.LOG_DAEMON,
		"LOG_AUTH":     syslog.LOG_AUTH,
		"LOG_SYSLOG":   syslog.LOG_SYSLOG,
		"LOG_LPR":      syslog.LOG_LPR,
		"LOG_NEWS":     syslog.LOG_NEWS,
		"LOG_UUCP":     syslog.LOG_UUCP,
		"LOG_CRON":     syslog.LOG_CRON,
		"LOG_AUTHPRIV": syslog.LOG_AUTHPRIV,
		"LOG_FTP":      syslog.LOG_FTP,
		"LOG_LOCAL0":   syslog.LOG_LOCAL0,
		"LOG_LOCAL1":   syslog.LOG_LOCAL1,
		"LOG_LOCAL2":   syslog.LOG_LOCAL2,
		"LOG_LOCAL3":   syslog.LOG_LOCAL3,
		"LOG_LOCAL4":   syslog.LOG_LOCAL4,
		"LOG_LOCAL5":   syslog.LOG_LOCAL5,
		"LOG_LOCAL6":   syslog.LOG_LOCAL6,
		"LOG_LOCAL7":   syslog.LOG_LOCAL7,
	}
}

var lSyslog *syslog.Writer

func Log() *syslog.Writer {
	if lSyslog == nil {
		l, err := syslog.New(syslog.LOG_DEBUG, "rrdserver")
		if err != nil {
			panic(err.Error())
		}
		lSyslog = l
	}
	return lSyslog
}

func Info(format string, args ...interface{}) {
	fmt.Println("Info: " + fmt.Sprintf(format, args...))
	Log().Info(fmt.Sprintf("Info: "+format, args...))
}

func Warning(format string, args ...interface{}) {
	fmt.Println("Warning: " + fmt.Sprintf(format, args...))
	Log().Warning("Warning: " + fmt.Sprintf(format, args...))
}

func Error(format string, args ...interface{}) {
	fmt.Printf("Error: " + fmt.Sprintf(format, args...))
	Log().Err("Error: " + fmt.Sprintf(format, args...))
}

func Fatal(format string, args ...interface{}) {
	Error(format, args...)
	os.Exit(2)
}

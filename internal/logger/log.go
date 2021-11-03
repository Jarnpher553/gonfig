package logger

import (
	"fmt"
	"github.com/Jarnpher553/gonfig/internal/util/color"
	alog "github.com/lesismal/arpc/log"
	"log"
)

type XLogger struct {
	level int
	mod   string
}

func (l *XLogger) SetLevel(lvl int) {
	l.level = lvl
}

func (l *XLogger) SetModName(mod string) {
	l.mod = mod
}

func (l *XLogger) Debug(format string, v ...interface{}) {
	if alog.LevelDebug >= l.level {
		if l.mod != "" {
			log.Printf(fmt.Sprintf("[%s] [%s] %s", color.Green("DBG"), color.Green(l.mod), format), v...)

		} else {
			log.Printf(fmt.Sprintf("[%s] %s", color.Green("DBG"), format), v...)
		}
	}
}

func (l *XLogger) Info(format string, v ...interface{}) {
	if alog.LevelInfo >= l.level {
		if l.mod != "" {
			log.Printf(fmt.Sprintf("[%s] [%s] %s", color.Green("INF"), color.Green(l.mod), format), v...)

		} else {
			log.Printf(fmt.Sprintf("[%s] %s", color.Green("INF"), format), v...)
		}
	}
}

func (l *XLogger) Warn(format string, v ...interface{}) {
	if alog.LevelWarn >= l.level {
		if l.mod != "" {
			log.Printf(fmt.Sprintf("[%s] [%s] %s", color.Green("WRN"), color.Green(l.mod), format), v...)

		} else {
			log.Printf(fmt.Sprintf("[%s] %s", color.Green("WRN"), format), v...)
		}
	}
}

func (l *XLogger) Error(format string, v ...interface{}) {
	if alog.LevelError >= l.level {
		if l.mod != "" {
			log.Printf(fmt.Sprintf("[%s] [%s] %s", color.Green("ERR"), color.Green(l.mod), format), v...)

		} else {
			log.Printf(fmt.Sprintf("[%s] %s", color.Green("ERR"), format), v...)
		}
	}
}

func (l *XLogger) Fatal(format string, v ...interface{}) {
	if l.mod != "" {
		log.Printf(fmt.Sprintf("[%s] [%s] %s", color.Green("FATAL"), color.Green(l.mod), format), v...)

	} else {
		log.Printf(fmt.Sprintf("[%s] %s", color.Green("FATAL"), format), v...)
	}
}

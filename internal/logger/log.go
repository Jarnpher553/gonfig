package logger

import (
	"fmt"
	"github.com/Jarnpher553/gonfig/internal/utility/color"
	alog "github.com/lesismal/arpc/log"
	"log"
)

type XLogger struct {
	level int
}

func (l *XLogger) SetLevel(lvl int) {
	l.level = lvl
}

func (l *XLogger) Debug(format string, v ...interface{}) {
	if alog.LevelDebug >= l.level {
		log.Printf(fmt.Sprintf("[%s] %s", color.Green("DBG"), format), v...)
	}
}

func (l *XLogger) Info(format string, v ...interface{}) {
	if alog.LevelInfo >= l.level {
		log.Printf(fmt.Sprintf("[%s] %s", color.Green("INF"), format), v...)
	}
}

func (l *XLogger) Warn(format string, v ...interface{}) {
	if alog.LevelWarn >= l.level {
		log.Printf(fmt.Sprintf("[%s] %s", color.Green("WRN"), format), v...)
	}
}

func (l *XLogger) Error(format string, v ...interface{}) {
	if alog.LevelError >= l.level {
		log.Printf(fmt.Sprintf("[%s] %s", color.Green("ERR"), format), v...)
	}
}

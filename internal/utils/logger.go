package logger

import (
	"fmt"
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
	"io"
	"os"
)

func NewLogger(AsCLI bool) *logrus.Logger {
	l := &logrus.Logger{
		Out:   os.Stderr,
		Level: logrus.DebugLevel,
		Formatter: &nested.Formatter{
			HideKeys:        true,
			NoColors:        true,
			TimestampFormat: "2006-01-02 15:04:05",
			FieldsOrder:     []string{"workflowId", "level", "message"},
		},
	}

	filename := "rocket.log"
	if os.Getenv("ROCKET_LOG_PATH") == "" {
		os.Setenv("ROCKET_LOG_PATH", filename)
	}
	f, err := os.OpenFile(os.Getenv("ROCKET_LOG_PATH"), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v", err)
	}
	if AsCLI == true {
		mw := io.MultiWriter(os.Stdout, f)
		l.SetOutput(mw)
	} else {
		l.SetOutput(f)
	}
	l.SetFormatter(&logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
			logrus.FieldKeyFunc:  "caller",
		},
	})
	return l
}

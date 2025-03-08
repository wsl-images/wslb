package logger

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

var (
	stdoutLog = logrus.New()
	fileLog   = logrus.New()
)

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		stdoutLog.Fatal("Unable to get user home directory: ", err)
	}

	logDir := filepath.Join(homeDir, ".wslb", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		stdoutLog.Fatal("Unable to create log directory: ", err)
	}

	logFile, err := os.OpenFile(
		filepath.Join(logDir, "wslb.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		stdoutLog.Fatal("Unable to open log file: ", err)
	}

	stdoutLog.SetOutput(os.Stdout)
	stdoutLog.SetFormatter(&logrus.TextFormatter{
		ForceColors:      true,
		FullTimestamp:    true,
		TimestampFormat:  "2006-01-02 15:04:05",
		DisableTimestamp: false,
	})

	fileLog.SetOutput(logFile)
	fileLog.SetFormatter(&logrus.TextFormatter{
		ForceColors:      false,
		DisableColors:    true,
		FullTimestamp:    true,
		TimestampFormat:  "2006-01-02 15:04:05",
		DisableTimestamp: false,
	})

	stdoutLog.SetLevel(logrus.InfoLevel)
	fileLog.SetLevel(logrus.InfoLevel)
}

func Info(args ...interface{}) {
	stdoutLog.Info(args...)
	fileLog.Info(args...)
}

func Error(args ...interface{}) {
	stdoutLog.Error(args...)
	fileLog.Error(args...)
}

func Fatal(args ...interface{}) {
	fileLog.Fatal(args...)
	stdoutLog.Fatal(args...)
}

func Debug(args ...interface{}) {
	stdoutLog.Debug(args...)
	fileLog.Debug(args...)
}

func Warn(args ...interface{}) {
	stdoutLog.Warn(args...)
	fileLog.Warn(args...)
}

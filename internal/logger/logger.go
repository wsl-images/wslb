package logger

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

type CustomTextFormatter struct {
	*logrus.TextFormatter
}

// Format overrides the TextFormatter Format method to skip level output
func (f *CustomTextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// Get the standard formatting
	b, err := f.TextFormatter.Format(entry)
	if err != nil {
		return nil, err
	}

	// For INFO level logs, remove the "INFO" prefix
	if entry.Level == logrus.InfoLevel {
		// Find the start of the message by skipping "INFO" and space
		parts := bytes.SplitN(b, []byte(" "), 2)
		if len(parts) == 2 {
			return parts[1], nil
		}
	}

	if entry.Level == logrus.ErrorLevel {
		// Find the start of the message by skipping "INFO" and space
		parts := bytes.SplitN(b, []byte(" "), 2)
		if len(parts) == 2 {
			return parts[1], nil
		}
	}

	return b, nil
}

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

	baseFormatter := &logrus.TextFormatter{
		ForceColors:      true,
		FullTimestamp:    false,
		TimestampFormat:  "2006-01-02 15:04:05",
		DisableTimestamp: true,
	}

	stdoutLog.SetOutput(os.Stdout)
	stdoutLog.SetOutput(os.Stdout)
	stdoutLog.SetFormatter(&CustomTextFormatter{
		TextFormatter: baseFormatter,
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

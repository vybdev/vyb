package logging

import (
	"github.com/sirupsen/logrus"
	"os"
)

var (
	// Log is the default logger for the application.
	Log = logrus.New()
)

// Init initializes the logger with the given log level.
func Init(level string) error {
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}

	Log.SetLevel(logLevel)
	Log.SetOutput(os.Stderr)
	Log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	return nil
}

package helpers

import (
	log "github.com/sirupsen/logrus"
	"github.com/snowzach/rotatefilehook"
	"os"
	"path"
	"time"
)

// InitLoggers sets the default logger options
func InitLoggers(level log.Level) {
	rotateFileHook, err := rotatefilehook.NewRotateFileHook(rotatefilehook.RotateFileConfig{
		Filename:   path.Join(LogDir(), "olaris-server.log"),
		MaxSize:    2,  // megabytes
		MaxBackups: 7,  // amount
		MaxAge:     28, //days
		Level:      log.DebugLevel,
		Formatter: &log.JSONFormatter{
			TimestampFormat: time.RFC822,
		},
	})
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Warnln("Could not setup logfile.")
	}
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{ForceColors: true})
	log.SetLevel(level)
	log.AddHook(rotateFileHook)
}

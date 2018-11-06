package helpers

import (
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"path"
)

// InitLoggers sets the default logger options
func InitLoggers(level log.Level) {
	f, err := os.OpenFile(path.Join(LogPath(), "olaris-server.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Warnln("Tried opening logfile for writing but got an error instead. Only logging to stdout.")
		log.SetOutput(os.Stdout)
	} else {
		mw := io.MultiWriter(os.Stdout, f)
		log.SetOutput(mw)
	}

	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(level)
}

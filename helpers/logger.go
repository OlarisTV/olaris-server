package helpers

import (
	log "github.com/sirupsen/logrus"
	"os"
)

func InitLoggers() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

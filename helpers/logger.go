package helpers

import (
	log "github.com/sirupsen/logrus"
)

func InitLoggers() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log.DebugLevel)
}

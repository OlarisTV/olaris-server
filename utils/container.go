package utils

import (
	"github.com/goava/di"
	log "github.com/sirupsen/logrus"
)

func MustResolve(c *di.Container, into interface{}, options ...di.ResolveOption) {
	err := c.Resolve(into, options...)
	if err != nil {
		log.Fatal(err)
	}
}

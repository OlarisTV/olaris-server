package helpers

import (
	log "github.com/sirupsen/logrus"
)

var refreshMDLocks = make(map[string]bool)

// WithLock keeps track of metaitems that are being refreshed based on an id or uuid and locks them so they can't be refreshed twice.
func WithLock(fn func(), id string) {
	log.WithFields(log.Fields{"lockID": id}).Debugln("Checking for lock.")
	if refreshMDLocks[id] == false {
		log.WithFields(log.Fields{"lockID": id}).Debugln("No lock.")
		refreshMDLocks[id] = true
		fn()
		refreshMDLocks[id] = false
		log.WithFields(log.Fields{"lockID": id}).Debugln("Lock released.")
	} else {
		log.WithFields(log.Fields{"lockID": id}).Warnln("Already had a lock, ignoring.")
	}
}

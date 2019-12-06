package helpers

import (
	log "github.com/sirupsen/logrus"
	"sync"
)

var mutex = &sync.Mutex{}
var refreshMDLocks = make(map[string]bool)

// WithLock keeps track of metaitems that are being refreshed based on an id or uuid and locks them so they can't be refreshed twice.
func WithLock(fn func(), id string) {
	log.WithFields(log.Fields{"lockID": id}).Debugln("Checking for lock.")
	mutex.Lock()
	if _, exists := refreshMDLocks[id]; !exists {
		log.WithFields(log.Fields{"lockID": id}).Debugln("No lock found, locking...")
		refreshMDLocks[id] = true
		log.WithFields(log.Fields{"lockID": id}).Debugln("Locked.")

		// No need to hold the mutex while fn() runs, so release it before we need it again
		mutex.Unlock()
		fn()
		mutex.Lock()
		delete(refreshMDLocks, id)
	} else {
		log.WithFields(log.Fields{"lockID": id}).Warnln("Already had a lock, ignoring.")
	}
	mutex.Unlock()
	log.WithFields(log.Fields{"lockID": id}).Debugln("Lock released.")
}

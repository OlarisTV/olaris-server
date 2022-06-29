package managers

import (
	"runtime"

	"github.com/Jeffail/tunny"
	log "github.com/sirupsen/logrus"
)

// WorkerPool is a container for the various workers that a library needs
type WorkerPool struct {
	probePool *tunny.Pool
}

// Shutdown properly shuts down the WP
func (p *WorkerPool) Shutdown() {
	log.Debugln("Shutting down worker pool")
	p.probePool.Close()
	log.Debugln("Pool shut down")
}

// NewDefaultWorkerPool needs a description
func NewDefaultWorkerPool() *WorkerPool {
	p := &WorkerPool{}

	p.probePool = tunny.NewFunc(runtime.NumCPU()*2, func(payload interface{}) interface{} {
		log.Debugln("Current probe queue length:", p.probePool.QueueLength())
		if job, ok := payload.(*probeJob); ok {
			job.man.ProbeFile(job.node)
		} else {
			log.Warnln("Got a ProbeJob that couldn't be cast as such, refreshing library might fail.")
		}
		return nil
	})

	return p
}

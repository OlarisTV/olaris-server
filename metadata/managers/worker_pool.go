package managers

import (
	"github.com/Jeffail/tunny"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/metadata/agents"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

// LibrarySubscriber is an interface that can implement the various notifications the libraryManager can give off
type LibrarySubscriber interface {
	MovieAdded(*db.Movie)
	EpisodeAdded(*db.Episode)
}

// WorkerPool is a container for the various workers that a library needs
type WorkerPool struct {
	tmdbPool   *tunny.Pool
	probePool  *tunny.Pool
	Subscriber LibrarySubscriber
}

// NewDefaultWorkerPool needs a description
func NewDefaultWorkerPool() *WorkerPool {
	p := &WorkerPool{}
	agent := agents.NewTmdbAgent()

	//TODO: We probably want a more global pool.
	// The MovieDB currently has a 40 requests per 10 seconds limit. Assuming every request takes a second then four workers is probably ideal.
	p.tmdbPool = tunny.NewFunc(4, func(payload interface{}) interface{} {
		log.Println("Current TMDB queue length:", p.tmdbPool.QueueLength())
		ep, ok := payload.(episodePayload)
		if ok {
			err := agents.UpdateEpisodeMD(agent, &ep.episode, &ep.season, &ep.series)
			if err != nil {
				log.WithFields(log.Fields{"error": err}).Warnln("Got an error updating metadata for series.")
			} else {
				db.UpdateEpisode(&ep.episode)
				if p.Subscriber != nil {
					log.Warnln("GIVING AN UPDATE TO THE NOTIFIER")
					p.Subscriber.EpisodeAdded(&ep.episode)
				}
			}
		}
		ok = false
		movie, ok := payload.(db.Movie)
		if ok {
			err := agents.UpdateMovieMD(agent, &movie)
			if err != nil {
				log.WithFields(log.Fields{"error": err}).Warnln("Got an error updating metadata for movie.")
			} else {
				db.UpdateMovie(&movie)
				db.MergeDuplicateMovies()
				if p.Subscriber != nil {
					log.Warnln("GIVING AN UPDATE TO THE NOTIFIER")
					p.Subscriber.MovieAdded(&movie)
				}
			}
		}

		return nil
	})

	p.probePool = tunny.NewFunc(4, func(payload interface{}) interface{} {
		log.Println("Current Probe queue length:", p.probePool.QueueLength())
		job, ok := payload.(probeJob)
		if ok {
			job.man.ProbeFile(job.node)
		}
		return nil
	})

	return p
}

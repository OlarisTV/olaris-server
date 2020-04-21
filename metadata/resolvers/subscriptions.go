package resolvers

import (
	"context"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"gitlab.com/olaris/olaris-server/metadata/managers/metadata"
	"time"
)

type eventFilterFn = func(e *metadata.MetadataEvent) bool

type metadataSubscription struct {
	eventFilterFn eventFilterFn
	metadataSubCh metadata.MetadataSubscriber

	stopCh    <-chan struct{}
	publishCh chan<- *MetadataEventResolver
}

func (s *metadataSubscription) Start() {
	for {
		select {
		case <-s.stopCh:
			return
		case e := <-s.metadataSubCh:
			if s.eventFilterFn(e) {
				// TODO(Leon Handreke): Warn about dropped events
				select {
				case s.publishCh <- ToEventResolver(e):
					// Empty, we already published the event to the client
				case <-time.After(2 * time.Second):
					log.WithFields(log.Fields{"event": e})
				}
			}
		}
	}
}

func ToEventResolver(e *metadata.MetadataEvent) *MetadataEventResolver {
	var r interface{}

	switch e.EventType {
	case metadata.MetadataEventTypeMovieAdded:
		r = &MovieAddedEventResolver{r: *e.Payload.(*db.Movie)}
	case metadata.MetadataEventTypeMovieUpdated:
		r = &MovieUpdatedEventResolver{r: *e.Payload.(*db.Movie)}
	case metadata.MetadataEventTypeMovieDeleted:
		r = &MovieDeletedEventResolver{r: *e.Payload.(*db.Movie)}
	case metadata.MetadataEventTypeEpisodeAdded:
		r = &EpisodeAddedEventResolver{r: *e.Payload.(*db.Episode)}
	case metadata.MetadataEventTypeEpisodeDeleted:
		r = &EpisodeDeletedEventResolver{r: *e.Payload.(*db.Episode)}

	case metadata.MetadataEventTypeSeasonAdded:
		r = &SeasonAddedEventResolver{r: *e.Payload.(*db.Season)}
	case metadata.MetadataEventTypeSeasonDeleted:
		r = &SeasonDeletedEventResolver{r: *e.Payload.(*db.Season)}

	case metadata.MetadataEventTypeSeriesAdded:
		r = &SeriesAddedEventResolver{r: *e.Payload.(*db.Series)}
	case metadata.MetadataEventTypeSeriesDeleted:
		r = &SeriesDeletedEventResolver{r: *e.Payload.(*db.Series)}
	default:
		panic("Failed to convert MetadataEvent to resolver.")

	}

	return &MetadataEventResolver{r: r}
}

func (r *Resolver) startMetadataSubscription(
	ctx context.Context,
	eventFilterFn eventFilterFn) <-chan *MetadataEventResolver {

	publishCh := make(chan *MetadataEventResolver, 10)
	subscription := metadataSubscription{
		eventFilterFn: eventFilterFn,
		metadataSubCh: r.env.MetadataManager.AddSubscriber(),
		stopCh:        ctx.Done(),
		publishCh:     publishCh,
	}
	go subscription.Start()

	return publishCh
}

func (r *Resolver) MoviesChanged(ctx context.Context) <-chan *MetadataEventResolver {
	log.Debugln("Adding subscription to Movies")
	return r.startMetadataSubscription(
		ctx,
		func(e *metadata.MetadataEvent) bool {
			if e.EventType == metadata.MetadataEventTypeMovieAdded ||
				e.EventType == metadata.MetadataEventTypeMovieUpdated ||
				e.EventType == metadata.MetadataEventTypeMovieDeleted {
				return true

			}
			return false
		})
}

func (r *Resolver) SeriesChanged(ctx context.Context) <-chan *MetadataEventResolver {
	log.Debugln("Adding subscription to Series")
	return r.startMetadataSubscription(
		ctx,
		func(e *metadata.MetadataEvent) bool {
			if e.EventType == metadata.MetadataEventTypeSeriesAdded ||
				e.EventType == metadata.MetadataEventTypeSeriesUpdated ||
				e.EventType == metadata.MetadataEventTypeSeriesDeleted {
				return true

			}
			return false
		})
}

type seasonChangedArgs struct {
	SeriesUUID *string
}

func (r *Resolver) SeasonChanged(
	ctx context.Context,
	args *seasonChangedArgs) (<-chan *MetadataEventResolver, error) {

	log.Debugln("Adding subscription to Series")
	// We need to find out the ID (not the UUID) here because by the time the event arrives,
	// the Series matching the Season of the arriving event may already be deleted,
	// so we can only filter the event by the (possibly dangling) ID reference in the Season object.
	// Also, it's just a lot faster, saves a DB lookup on every event.
	series, err := db.FindSeriesByUUID(*args.SeriesUUID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to find series")
	}
	seriesID := series.ID

	return r.startMetadataSubscription(
		ctx,
		func(e *metadata.MetadataEvent) bool {
			if e.EventType == metadata.MetadataEventTypeSeasonAdded ||
				e.EventType == metadata.MetadataEventTypeSeasonUpdated ||
				e.EventType == metadata.MetadataEventTypeSeasonDeleted {

				season := e.Payload.(*db.Season)
				if season.SeriesID == seriesID {
					return true
				}
			}
			return false
		}), nil
}

type MetadataEventResolver struct {
	r interface{}
}

func (r *MetadataEventResolver) ToMovieAddedEvent() (*MovieAddedEventResolver, bool) {
	res, ok := r.r.(*MovieAddedEventResolver)
	return res, ok
}

func (r *MetadataEventResolver) ToMovieUpdatedEvent() (*MovieUpdatedEventResolver, bool) {
	res, ok := r.r.(*MovieUpdatedEventResolver)
	return res, ok
}

func (r *MetadataEventResolver) ToMovieDeletedEvent() (*MovieDeletedEventResolver, bool) {
	res, ok := r.r.(*MovieDeletedEventResolver)
	return res, ok
}

func (r *MetadataEventResolver) ToSeriesAddedEvent() (*SeriesAddedEventResolver, bool) {
	res, ok := r.r.(*SeriesAddedEventResolver)
	return res, ok
}

func (r *MetadataEventResolver) ToSeriesDeletedEvent() (*SeriesDeletedEventResolver, bool) {
	res, ok := r.r.(*SeriesDeletedEventResolver)
	return res, ok
}

func (r *MetadataEventResolver) ToSeasonAddedEvent() (*SeasonAddedEventResolver, bool) {
	res, ok := r.r.(*SeasonAddedEventResolver)
	return res, ok
}

func (r *MetadataEventResolver) ToSeasonDeletedEvent() (*SeasonDeletedEventResolver, bool) {
	res, ok := r.r.(*SeasonDeletedEventResolver)
	return res, ok
}

func (r *MetadataEventResolver) ToEpisodeAddedEvent() (*EpisodeAddedEventResolver, bool) {
	res, ok := r.r.(*EpisodeAddedEventResolver)
	return res, ok
}

func (r *MetadataEventResolver) ToEpisodeDeletedEvent() (*EpisodeDeletedEventResolver, bool) {
	res, ok := r.r.(*EpisodeDeletedEventResolver)
	return res, ok
}

type MovieAddedEventResolver struct {
	r db.Movie
}

func (r *MovieAddedEventResolver) Movie() *MovieResolver {
	return &MovieResolver{r.r}
}

type MovieUpdatedEventResolver struct {
	r db.Movie
}

func (r *MovieUpdatedEventResolver) Movie() *MovieResolver {
	return &MovieResolver{r.r}
}

type MovieDeletedEventResolver struct {
	r db.Movie
}

func (r *MovieDeletedEventResolver) MovieUUID() string {
	return r.r.UUID
}

type SeriesAddedEventResolver struct {
	r db.Series
}

func (r *SeriesAddedEventResolver) Series() *SeriesResolver {
	return &SeriesResolver{r.r}
}

type SeriesDeletedEventResolver struct {
	r db.Series
}

func (r *SeriesDeletedEventResolver) SeriesUUID() string {
	return r.r.UUID
}

type SeasonAddedEventResolver struct {
	r db.Season
}

func (r *SeasonAddedEventResolver) Season() *SeasonResolver {
	return &SeasonResolver{r.r}
}

type SeasonDeletedEventResolver struct {
	r db.Season
}

func (r *SeasonDeletedEventResolver) SeasonUUID() string {
	return r.r.UUID
}

type EpisodeAddedEventResolver struct {
	r db.Episode
}

func (r *EpisodeAddedEventResolver) Episode() *EpisodeResolver {
	return &EpisodeResolver{r.r}
}

type EpisodeDeletedEventResolver struct {
	r db.Episode
}

func (r *EpisodeDeletedEventResolver) EpisodeUUID() string {
	return r.r.UUID
}

func warnDroppedEvent(m string, n string) {
	log.WithFields(log.Fields{"eventType": m, "name": n}).Warnln("Subscription event could not be pushed into channel. Events might be missed.")
}

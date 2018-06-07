package resolvers

import (
	"fmt"
	"github.com/graph-gophers/graphql-go/relay"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
	"net/http"
)

type Resolver struct {
	ctx *db.MetadataContext
}

type LibraryResolver struct {
	r Library
}

func (r *LibraryResolver) Name() string {
	return r.r.Name
}
func (r *LibraryResolver) Movies() []*MovieResolver {
	return r.r.Movies
}
func (r *LibraryResolver) Episodes() []*EpisodeResolver {
	return r.r.Episodes
}
func (r *LibraryResolver) FilePath() string {
	return r.r.FilePath
}
func (r *LibraryResolver) Kind() int32 {
	return int32(r.r.Kind)
}

type CreateLibraryArgs struct {
	Name     string
	FilePath string
	Kind     int32
}

func (r *Resolver) CreateLibrary(args *CreateLibraryArgs) *libResResolv {
	library, err := db.AddLibrary(args.Name, args.FilePath, db.MediaType(args.Kind))
	var libRes LibRes
	if err == nil {
		fmt.Println("Scaninng library")
		go db.NewLibraryManager(nil).RefreshAll()
		libRes = LibRes{Error: &ErrorResolver{Error{hasError: false}}, Library: &LibraryResolver{Library{library, nil, nil}}}
	} else {
		libRes = LibRes{Error: &ErrorResolver{Error{hasError: true, message: err.Error()}}, Library: &LibraryResolver{Library{}}}
	}
	return &libResResolv{libRes}
}

type libResResolv struct {
	r LibRes
}

func (r *libResResolv) Library() *LibraryResolver {
	return r.r.Library
}
func (r *libResResolv) Error() *ErrorResolver {
	return r.r.Error
}

type ErrorResolver struct {
	r Error
}

type LibRes struct {
	Error   *ErrorResolver
	Library *LibraryResolver
}

type Error struct {
	message  string
	hasError bool
}

func CreateErr(err error) Error {
	return Error{message: err.Error(), hasError: true}
}

func CreateErrResolver(err error) *ErrorResolver {
	return &ErrorResolver{r: Error{message: err.Error(), hasError: true}}
}

func (r *ErrorResolver) Message() string {
	return r.r.message
}
func (r *ErrorResolver) HasError() bool {
	return r.r.hasError
}

//mutation AddLibrary($name: String!, $file_path: String!, $kind: Int!) {
//	createLibrary(name: $name, file_path: $file_path, kind: $kind){
//    library{
//      name
//    }
//    error{
//      hasError
//      message
//    }
//  }
//}

func GraphiQLHandler(w http.ResponseWriter, r *http.Request) {
	w.Write(graphiQLpage)
}

func NewRelayHandler(ctx *db.MetadataContext) *relay.Handler {
	schema := InitSchema(ctx)
	return &relay.Handler{Schema: schema}
}

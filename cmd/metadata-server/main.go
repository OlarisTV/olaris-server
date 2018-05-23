package main

import (
	"flag"
	"fmt"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/ryanbradynd05/go-tmdb"
	"gitlab.com/bytesized/bytesized-streaming/metadata"
	"log"
	"net/http"
	"os/user"
	"path"
)

var mediaFilesDir = flag.String("media_files_dir", "/var/media", "Path used if no libraries exist for the default library")

func main() {
	flag.Parse()

	usr, err := user.Current()
	if err != nil {
		fmt.Println("Failed to determine user's home directory: ", err.Error())
	}
	dbPath := path.Join(usr.HomeDir, ".config", "bss", "metadb")
	metadata.EnsurePath(dbPath)
	db, err := gorm.Open("sqlite3", path.Join(dbPath, "bsmdb_data.db"))
	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %s\n", err))
	}
	defer db.Close()

	// Migrate the schema
	db.AutoMigrate(&metadata.MovieItem{}, &metadata.Library{})
	schema := metadata.InitSchema(db)

	apiKey := "0cdacd9ab172ac6ff69c8d84b2c938a8"
	tmdb := tmdb.Init(apiKey)

	libraryManager := metadata.NewLibraryManager(db, tmdb)

	var count int
	db.Table("libraries").Count(&count)
	if count == 0 {
		fmt.Println("Saving initial movie library")
		libraryManager.AddLibrary("Movies", *mediaFilesDir)
	}

	libraryManager.ActivateAll()

	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(page)
	}))

	http.Handle("/query", &relay.Handler{Schema: schema})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

var page = []byte(`
<!DOCTYPE html>
<html>
	<head>
		<link href="https://cdnjs.cloudflare.com/ajax/libs/graphiql/0.11.11/graphiql.min.css" rel="stylesheet" />
		<script src="https://cdnjs.cloudflare.com/ajax/libs/es6-promise/4.1.1/es6-promise.auto.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/fetch/2.0.3/fetch.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/react/16.2.0/umd/react.production.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/react-dom/16.2.0/umd/react-dom.production.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/graphiql/0.11.11/graphiql.min.js"></script>
	</head>
	<body style="width: 100%; height: 100%; margin: 0; overflow: hidden;">
		<div id="graphiql" style="height: 100vh;">Loading...</div>
		<script>
			function graphQLFetcher(graphQLParams) {
				return fetch("/query", {
					method: "post",
					body: JSON.stringify(graphQLParams),
					credentials: "include",
				}).then(function (response) {
					return response.text();
				}).then(function (responseBody) {
					try {
						return JSON.parse(responseBody);
					} catch (error) {
						return responseBody;
					}
				});
			}
			ReactDOM.render(
				React.createElement(GraphiQL, {fetcher: graphQLFetcher}),
				document.getElementById("graphiql")
			);
		</script>
	</body>
</html>
`)

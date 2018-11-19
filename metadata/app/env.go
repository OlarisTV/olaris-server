// Package app wraps all other important packages.
package app

import (
	"database/sql/driver"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/jasonlvhit/gocron"
	"github.com/jinzhu/gorm"
	"math/rand"
	"reflect"
	"regexp"
	"time"
	// Import sqlite dialect
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/helpers"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"gitlab.com/olaris/olaris-server/metadata/managers"
	"strings"
)

// MetadataContext is a container for all important vars.
type MetadataContext struct {
	Db             *gorm.DB
	Watcher        *fsnotify.Watcher
	LibraryManager *managers.LibraryManager
	ExitChan       chan int
}

var env *MetadataContext

// GormLogger ensures logging for db queries uses logrus
type GormLogger struct{}

var sqlRegexp = regexp.MustCompile(`(\$\d+)|\?`)

// Print ensures the db logs as default logrus
func (l *GormLogger) Print(values ...interface{}) {
	entry := log.WithField("name", "database")
	if len(values) > 1 {
		level := values[0]
		source := values[1]
		entry = log.WithField("source", source)
		if level == "sql" {
			duration := values[2]
			// sql
			var formattedValues []interface{}
			for _, value := range values[4].([]interface{}) {
				indirectValue := reflect.Indirect(reflect.ValueOf(value))
				if indirectValue.IsValid() {
					value = indirectValue.Interface()
					if t, ok := value.(time.Time); ok {
						formattedValues = append(formattedValues, fmt.Sprintf("'%v'", t.Format(time.RFC3339)))
					} else if b, ok := value.([]byte); ok {
						formattedValues = append(formattedValues, fmt.Sprintf("'%v'", string(b)))
					} else if r, ok := value.(driver.Valuer); ok {
						if value, err := r.Value(); err == nil && value != nil {
							formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
						} else {
							formattedValues = append(formattedValues, "NULL")
						}
					} else {
						formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
					}
				} else {
					formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
				}
			}
			entry.WithField("took", duration).Debug(fmt.Sprintf(sqlRegexp.ReplaceAllString(values[3].(string), "%v"), formattedValues...))
		} else {
			entry.Error(values[2:]...)
		}
	} else {
		entry.Error(values...)
	}

}

// NewDefaultMDContext creates a new env with sane defaults.
func NewDefaultMDContext() *MetadataContext {
	dbPath := helpers.MetadataConfigPath()
	return NewMDContext(dbPath, true, true)
}

// NewMDContext lets you create a more custom environment.
func NewMDContext(dbPath string, dbLogMode bool, verboseLog bool) *MetadataContext {
	rand.Seed(time.Now().UTC().UnixNano())

	logLevel := log.InfoLevel
	if verboseLog == true {
		logLevel = log.DebugLevel
	}

	helpers.InitLoggers(logLevel)

	log.Printf("olaris metadata server - v%s", helpers.Version())

	db := db.NewDb(dbPath, dbLogMode)
	db.SetLogger(&GormLogger{})

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(fmt.Sprintf("Could not start filesystem watcher: %s\n", err))
	}

	exitChan := make(chan int)

	env = &MetadataContext{Db: db, Watcher: watcher, ExitChan: exitChan}

	libraryManager := managers.NewLibraryManager(watcher)
	env.LibraryManager = libraryManager

	// Scan once on start-up
	go libraryManager.RefreshAll()

	go env.startWatcher(exitChan)

	gocron.Every(2).Hours().Do(managers.RefreshAgentMetadataWithMissingArt)
	gocron.Start()

	return env
}

func (env *MetadataContext) startWatcher(exitChan chan int) {
	log.Println("Starting fsnotify watchers.")
loop:
	for {
		select {
		case <-exitChan:
			log.Println("Stopping fsnotify watchers.")
			env.Watcher.Close()
			break loop
		case event := <-env.Watcher.Events:
			log.WithFields(log.Fields{"filename": event.Name, "event": event.Op}).Debugln("Got filesystem notification event.")
			if managers.ValidFile(event.Name) {
				if event.Op&fsnotify.Rename == fsnotify.Rename {
					log.Debugln("File is renamed, forcing removed files scan.")
					env.LibraryManager.CheckRemovedFiles() // Make this faster by only scanning the changed file
				}

				if event.Op&fsnotify.Remove == fsnotify.Remove {
					log.Debugln("File is removed, forcing removed files scan and removing fsnotify watch.")
					env.Watcher.Remove(event.Name)
					env.LibraryManager.CheckRemovedFiles() // Make this faster by only scanning the changed file
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					log.Debugln("File added was added adding watcher and requestin library rescan.")
					env.Watcher.Add(event.Name)
					for _, lib := range db.AllLibraries() {
						if strings.Contains(event.Name, lib.FilePath) {
							env.LibraryManager.ProbeFile(&lib, event.Name)
							// We can probably only get the MD for the recently added file here
							env.LibraryManager.UpdateMD(&lib)
						}
					}
				}
			} else {
				log.WithFields(log.Fields{"filename": event.Name, "event": event.Op}).Debugln("Got an error while trying to open file. Going to assume the file was removed.")
				log.Debugln("File is removed, forcing removed files scan and removing fsnotify watch.")
				env.Watcher.Remove(event.Name)
				env.LibraryManager.CheckRemovedFiles() // Make this faster by only scanning the changed file
			}
		case err := <-env.Watcher.Errors:
			log.Warnln("fsnotify watcher error:", err)
		}
	}
}

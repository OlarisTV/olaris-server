package main

import (
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/metadata/auth"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"time"
)

var username = flag.String("username", "", "User to generate a token for")

func main() {
	flag.Parse()

	mctx := app.NewDefaultMDContext()
	defer mctx.Db.Close()

	user, err := db.FindUserByUsername(*username)
	if err != nil {
		log.Fatalf("Failed to find user \"%s\": %s", *username, err.Error())
	}

	// Create a token quasi-unlimited validity.
	jwt, err := auth.CreateMetadataJWT(user, 1000*24*time.Hour)
	if err != nil {
		log.Fatalf("Failed to create login token: %s", err.Error())
	}

	fmt.Println("Bearer", jwt)
}

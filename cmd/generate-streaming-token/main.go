package main

import (
	"flag"
	"fmt"
	"gitlab.com/olaris/olaris-server/metadata/auth"
	"os"
)

var filepath = flag.String("filepath", "", "Filepath to generate a token for")

func main() {
	flag.Parse()

	if *filepath == "" {
		fmt.Println("Must supply a filepath")
		os.Exit(1)

	}

	token, err := auth.CreateStreamingJWT(0, *filepath)
	if err != nil {
		fmt.Printf("Failed to create streaming token: %s", err.Error())
		os.Exit(1)
	}
	fmt.Println(token)
}

package main

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	"gitlab.com/bytesized/bytesized-streaming/metadata/auth"
)

var filepath = flag.String("filepath", "", "Filepath to generate a token for")

func main() {
	flag.Parse()

	if *filepath == "" {
		glog.Exit("Must supply a filepath")
	}

	token, err := auth.CreateStreamingJWT(0, *filepath)
	if err != nil {
		glog.Exitf("Failed to create streaming token: %s", err.Error())
	}
	fmt.Println(token)
}

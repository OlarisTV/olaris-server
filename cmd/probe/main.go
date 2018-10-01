package main

import (
	"fmt"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"os"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Println("please supply a filename")
		return
	}

	probe, err := ffmpeg.Probe(os.Args[1])
	if err != nil {
		fmt.Println("ERROR:", err)
		return
	}

	for _, s := range probe.Streams {
		fmt.Printf("%s\n", s.String())
	}
}

package main

import (
	"fmt"
	"gitlab.com/bytesized/bytesized-streaming/dash"
	"os"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Println("please supply a filename")
		return
	}
	manifest := dash.BuildTransmuxingManifestFromFile(os.Args[1])
	fmt.Println(manifest)
}

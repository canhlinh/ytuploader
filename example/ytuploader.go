package main

import (
	"fmt"
	"log"
	"runtime"

	"github.com/canhlinh/ytuploader"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	cookies, err := ytuploader.ParseCookiesFromJSONFile("cookie.json")
	if err != nil {
		log.Fatal(err)
	}

	uploader := ytuploader.New(".")
	videoURL, err := uploader.Upload("", "./big_buck_bunny_720p_20mb.mp4", cookies, false)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(videoURL)
}

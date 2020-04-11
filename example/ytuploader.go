package main

import (
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

	uploader := ytuploader.New(false, ".")
	defer uploader.Stop()

	videoURL, err := uploader.Upload("", "./big_buck_bunny_720p_20mb.mp4", cookies.Builtin(), false)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Upload completed ", videoURL)
}

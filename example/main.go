package main

import (
	"fmt"
	"log"
	"runtime"

	"github.com/canhlinh/ytuploader"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	ytuploader.DefaultChromedriverPort = 4444

	cookies, err := ytuploader.ParseCookiesFromJSONFile("cookie.json")
	if err != nil {
		log.Fatal(err)
	}

	uploader := ytuploader.New(".", "someone@gmail.com")
	videoURL, err := uploader.Upload("", "./big_buck_bunny_720p_20mb.mp4", cookies.Builtin(), false)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(videoURL)
}

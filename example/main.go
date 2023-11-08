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

	uploader := ytuploader.New("./", "someone@gmail.com", ytuploader.DefaultUserAgent)
	uploader.Headless = false
	thumbnail := "sample.jpg"
	if videoURL, err := uploader.Upload("", "./big_buck_bunny_720p_20mb.mp4", cookies.Builtin(), &thumbnail, false); err != nil {
		log.Fatal(err)
	} else {
		fmt.Println(videoURL)
	}

	// uploader.Headless = true
	// if videoURL, err := uploader.Upload("", "./big_buck_bunny_720p_20mb.mp4", cookies.Builtin(), &thumbnail, false); err != nil {
	// 	log.Fatal(err)
	// } else {
	// 	fmt.Println(videoURL)
	// }
}

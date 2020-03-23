package main

import (
	"log"

	"github.com/canhlinh/ytuploader"
)

func main() {

	cookies, err := ytuploader.ParseCookiesFromJSONFile("cookie.json")
	if err != nil {
		log.Fatal(err)
	}

	uploader := ytuploader.New(false)
	videoURL, err := uploader.Upload("UC5zftEj8KAc92dPx_nOk9LA", "./sample.mov", cookies.Builtin())
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Upload completed ", videoURL)
}

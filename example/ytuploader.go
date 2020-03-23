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

	uploader := ytuploader.New(true, ".")
	videoURL, err := uploader.Upload("", "./sample.mov", cookies.Builtin(), false)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Upload completed ", videoURL)
}

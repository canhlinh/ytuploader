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

	uploader := ytuploader.New(".")
	videoURL, err := uploader.Upload("", "/home/kyonguyen/Downloads/Equine_Trot_400fps_Right.avi", cookies.Builtin(), false)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(videoURL)
}

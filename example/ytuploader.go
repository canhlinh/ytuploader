package main

import (
	"log"
	"runtime"
	"sync"

	"github.com/canhlinh/ytuploader"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	cookies, err := ytuploader.ParseCookiesFromJSONFile("cookie.json")
	if err != nil {
		log.Fatal(err)
	}

	wait := sync.WaitGroup{}
	wait.Add(5)

	for i := 0; i < 5; i++ {
		go func() {
			defer wait.Done()

			uploader := ytuploader.New(true, ".")
			defer uploader.Stop()

			videoURL, err := uploader.Upload("", "./big_buck_bunny_720p_20mb.mp4", cookies.Builtin(), false)
			if err != nil {
				log.Fatal(err)
			}
			log.Println("Upload completed ", videoURL)
		}()
	}

	wait.Wait()
}

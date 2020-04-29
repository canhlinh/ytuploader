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

	w := &sync.WaitGroup{}
	for i := 0; i < 2; i++ {
		threadNumber := i
		w.Add(1)
		go func() {
			defer w.Done()
			uploader := ytuploader.New(".")
			videoURL, err := uploader.Upload("", "./big_buck_bunny_720p_20mb.mp4", cookies.Builtin(), false)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Upload completed thread %d url %s\n", threadNumber, videoURL)
		}()
	}
	w.Wait()
}

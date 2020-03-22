package ytuploader

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/sclevine/agouti"
)

// YtUploader presents an uploader
type YtUploader struct {
	Driver *agouti.WebDriver
}

// New creates a new upload instance
func New(headless bool) *YtUploader {

	options := []agouti.Option{}
	if headless {
		options = append(options, agouti.ChromeOptions("args", []string{"--headless", "--disable-gpu", "--disable-crash-reporter"}))
	}

	driver := agouti.ChromeDriver(options...)

	if err := driver.Start(); err != nil {
		log.Fatal(err)
	}

	return &YtUploader{
		Driver: driver,
	}
}

// Upload uploads file to Youtube
func (ul *YtUploader) Upload(channel string, filepath string, cookies []*http.Cookie) (string, error) {
	page, err := ul.Driver.NewPage(agouti.Browser("chrome"))
	if err != nil {
		return "", err
	}
	defer page.CloseWindow()

	if err := page.Navigate("https://youtube.com"); err != nil {
		return "", err
	}

	for _, cookie := range cookies {
		if err := page.SetCookie(cookie); err != nil {
			return "", err
		}
	}

	uploadURL := "https://youtube.com/upload"
	uploadToChannel := false

	if channel != "" {
		uploadURL = fmt.Sprintf("https://studio.youtube.com/channel/%s", channel)
		uploadToChannel = true
	}

	if err := page.Navigate(uploadURL); err != nil {
		return "", err
	}

	if uploadToChannel {
		if err := page.FindByID("upload-icon").Click(); err != nil {
			return "", err
		}
	}

	if err := page.FindByXPath("//input[@name='Filedata']").UploadFile(filepath); err != nil {
		return "", err
	}

WAIT_SUBMIT:
	for {
		select {
		case <-time.NewTimer(time.Second * 3).C:
			return "", errors.New("File can't start upload. Timeout")
		default:
			if count, err := page.Find("a[class*='ytcp-video-metadata-info']").Count(); err == nil && count > 0 {
				log.Println("File in uploading")
				break WAIT_SUBMIT
			} else {
				log.Println("Waiting file submit")
				time.Sleep(time.Second)
			}
		}

	}

	for {

		value, err := page.Find(".progress.ytcp-uploads-dialog paper-progress.progress-container.style-scope.ytcp-video-upload-progress").Attribute("value")
		if err != nil {
			return "", err
		}

		percent, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return "", err
		}

		log.Printf("Uploaded %d percent\n", percent)
		if percent > 99 {
			break
		}
		time.Sleep(time.Second)
	}

	if err := page.FindByName("NOT_MADE_FOR_KIDS").Click(); err != nil {
		log.Fatal(err)
	}

	if err := page.FindByID("next-button").Click(); err != nil {
		log.Fatal(err)
	}

	if err := page.FindByID("next-button").Click(); err != nil {
		log.Fatal(err)
	}

	if err := page.FindByName("PRIVATE").Click(); err != nil {
		log.Fatal(err)
	}

	if err := page.FindByID("done-button").Click(); err != nil {
		log.Fatal(err)
	}

	videoURL, err := page.Find("a[class*='ytcp-video-metadata-info']").Attribute("href")
	if err != nil {
		return "", err
	}

	return videoURL, nil
}

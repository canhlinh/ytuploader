package ytuploader

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/sclevine/agouti"
)

// YtUploader presents an uploader
type YtUploader struct {
	Driver *agouti.WebDriver
}

// New creates a new upload instance
func New() *YtUploader {
	driver := agouti.ChromeDriver(
		agouti.ChromeOptions("args", []string{"--headless", "--disable-gpu", "--disable-crash-reporter"}),
	)

	if err := driver.Start(); err != nil {
		log.Fatal(err)
	}

	return &YtUploader{
		Driver: driver,
	}
}

// Upload uploads file to Youtube
func (ul *YtUploader) Upload(filepath string, cookies []*http.Cookie) (string, error) {
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

	page.Navigate("https://youtube.com/upload")
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

		percent, err := page.Find(".progress.ytcp-uploads-dialog paper-progress.progress-container.style-scope.ytcp-video-upload-progress").Attribute("value")
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Uploaded %s percent\n", percent)
		time.Sleep(time.Second)
		if percent == "100" {
			break
		}
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

	if err := page.FindByID("done-button").Click(); err != nil {
		log.Fatal(err)
	}

	videoURL, err := page.Find("a[class*='ytcp-video-metadata-info']").Attribute("href")
	if err != nil {
		return "", err
	}

	return videoURL, nil
}

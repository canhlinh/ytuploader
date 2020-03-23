package ytuploader

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/sclevine/agouti"
)

const (
	VideoProgressBoxClass = ".progress.ytcp-uploads-dialog paper-progress.progress-container.style-scope.ytcp-video-upload-progress"
)

// YtUploader presents an uploader
type YtUploader struct {
	Driver           *agouti.WebDriver
	screenshotFolder string
}

// New creates a new upload instance
func New(headless bool, screenshotFolder string) *YtUploader {

	options := []agouti.Option{}
	if headless {
		options = append(options,
			agouti.ChromeOptions(
				"args",
				[]string{
					"--headless",
					"--disable-gpu",
					"--no-sandbox",
					"--disable-crash-reporter",
				}),
		)
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
func (ul *YtUploader) Upload(channel string, filepath string, cookies []*http.Cookie, save bool) (string, error) {
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
		page.Screenshot("screenshot/error.png")
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

	timeout := time.NewTimer(time.Second * 5).C
WAIT_SUBMIT:
	for {
		select {
		case <-timeout:
			page.Screenshot(path.Join(ul.screenshotFolder, fmt.Sprintf("%d.png", time.Now().Unix())))
			return "", errors.New("File can't start upload. Timeout")
		default:
			if count, err := page.All(VideoProgressBoxClass).Count(); err == nil && count > 0 {
				log.Println("File in uploading")
				break WAIT_SUBMIT
			} else {
				log.Println("Waiting file submit")
				time.Sleep(time.Second)
			}
		}

	}

	uploadedPercent := int64(0)
	for {

		value, err := page.Find(VideoProgressBoxClass).Attribute("value")
		if err != nil {
			if uploadedPercent < 50 {
				return "", err
			}

			log.Println("Upload completed")
			break
		}

		uploadedPercent, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return "", err
		}

		log.Printf("Uploaded %d percent\n", uploadedPercent)
		if uploadedPercent >= 99 {
			log.Println("Upload completed")
			break
		}
		time.Sleep(time.Second)
	}

	if save {
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
	}

	videoURL, err := page.Find("a[class*='ytcp-video-metadata-info']").Attribute("href")
	if err != nil {
		return "", err
	}

	return videoURL, nil
}

// Stop stops the chromedrive instance
func (ul *YtUploader) Stop() {
	ul.Driver.Stop()
}

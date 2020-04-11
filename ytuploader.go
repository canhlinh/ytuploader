package ytuploader

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/sclevine/agouti"
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
					"--disable-setuid-sandbox",
				}),
		)
	}

	driver := agouti.ChromeDriver(options...)

	if err := driver.Start(); err != nil {
		log.Fatal(err)
	}

	return &YtUploader{
		Driver:           driver,
		screenshotFolder: screenshotFolder,
	}
}

// Upload uploads file to Youtube
func (ul *YtUploader) Upload(channel string, filepath string, cookies []*http.Cookie, save bool) (string, error) {

	page, err := ul.Driver.NewPage(agouti.Browser("chrome"))
	if err != nil {
		return "", err
	}

	defer page.CloseWindow()

	if err := page.Navigate("https://www.youtube.com"); err != nil {
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

	time.Sleep(time.Second * 2)

	if uploadToChannel {
		log.Println("Upload to channel")
		if count, err := page.AllByID("upload-button").Count(); err == nil && count > 0 {
			if err := page.FindByID("upload-button").Click(); err != nil {
				return "", err
			}
		} else {
			if err := page.FindByID("upload-icon").Click(); err != nil {
				return "", err
			}
		}

	}

	if _, err := os.Stat(filepath); err != nil {
		return "", err
	}

	if err := page.AllByXPath("//input[@type='file']").UploadFile(filepath); err != nil {
		return "", err
	}

	if err := waitFileSubmitting(page); err != nil {
		return "", err
	}

	percent := int64(0)
	for {

		value, err := page.Find(".progress-container.ytcp-video-upload-progress").Attribute("value")
		if err != nil {
			if percent < 95 {
				return "", err
			}

			log.Printf("Upload completed %d%%\n", percent)
			break
		}

		percent, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return "", err
		}

		log.Printf("Uploaded %d%%\n", percent)
		if percent == 100 {
			log.Printf("Upload completed %d%%\n", percent)
			break
		}
		time.Sleep(time.Second)
	}

	// Sleep 5 seconds to ensure the uploading progress is done.
	time.Sleep(time.Second * 5)

	if save {
		if err := page.FindByName("NOT_MADE_FOR_KIDS").Click(); err != nil {
			return "", err
		}

		if err := page.FindByID("next-button").Click(); err != nil {
			return "", err
		}

		if err := page.FindByID("next-button").Click(); err != nil {
			return "", err
		}

		if err := page.FindByID("done-button").Click(); err != nil {
			return "", err
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

func waitFileSubmitting(page *agouti.Page) error {
	timeout := time.NewTimer(time.Second * 5).C
	for {
		select {
		case <-timeout:
			return errors.New("File can't start upload. Timeout")
		default:
			if count, err := page.All("ytcp-uploads-details").Count(); err == nil && count > 0 {
				log.Println("File in uploading")
				return nil

			} else {
				log.Println("Waiting file submit")
				time.Sleep(time.Second)
			}
		}
	}
}

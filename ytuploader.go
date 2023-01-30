package ytuploader

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"path/filepath"

	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
)

// YtUploader presents an uploader
type YtUploader struct {
	screenshotFolder string
}

// New creates a new upload instance
func New(screenshotFolder string) *YtUploader {

	return &YtUploader{
		screenshotFolder: screenshotFolder,
	}
}

// Upload uploads file to Youtube
func (ul *YtUploader) Upload(channel string, filename string, cookies []*Cookie, save bool) (string, error) {
	service, err := selenium.NewChromeDriverService("chromedriver", 4444)
	if err != nil {
		return "", err
	}
	defer service.Stop()

	caps := selenium.Capabilities{}
	caps.AddChrome(chrome.Capabilities{Args: []string{
		"window-size=1920x1080",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--disable-gpu",
		// "--headless",  // comment out this line to see the browser
	}})

	driver, err := selenium.NewRemote(caps, "")
	if err != nil {
		return "", err
	}
	defer driver.Close()

	if err := driver.Get("https://www.youtube.com"); err != nil {
		return "", err
	}

	for _, cookie := range cookies {
		if err := driver.AddCookie(&selenium.Cookie{
			Name:   cookie.Name,
			Value:  cookie.Value,
			Path:   cookie.Path,
			Domain: cookie.Domain,
			Secure: cookie.Secure,
			Expiry: uint(cookie.ExpirationDate),
		}); err != nil {
			return "", err
		}
	}

	time.Sleep(time.Second * 1)

	uploadURL := "https://youtube.com/upload"
	uploadToChannel := false

	if channel != "" {
		uploadURL = fmt.Sprintf("https://studio.youtube.com/channel/%s", channel)
		uploadToChannel = true
	}

	if err := driver.Get(uploadURL); err != nil {
		return "", err
	}

	if uploadToChannel {
		log.Println("Upload to channel")
		button, err := driver.FindElement(selenium.ByID, "upload-button")
		if err != nil {
			button, err = driver.FindElement(selenium.ByID, "upload-icon")
			if err != nil {
				return "", err
			}
		}
		if err := button.Click(); err != nil {
			return "", err
		}
	}

	absFilePath, err := filepath.Abs(filename)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(absFilePath); err != nil {
		return "", err
	}

	element, err := driver.FindElement(selenium.ByXPATH, "//div[@id='content']/input")
	if err != nil {
		return "", err
	}

	if err := element.SendKeys(absFilePath); err != nil {
		return "", err
	}

	if err := driver.WaitWithTimeout(func(wd selenium.WebDriver) (bool, error) {
		_, err := wd.FindElement(selenium.ByCSSSelector, ".ytcp-uploads-dialog")
		return err == nil, err
	}, 3*time.Second); err != nil {
		return "", err
	}

	if save {
		if e, err := driver.FindElement(selenium.ByName, "NOT_MADE_FOR_KIDS"); err != nil {
			return "", err
		} else {
			e.Click()
		}

		if e, err := driver.FindElement(selenium.ByID, "next-button"); err != nil {
			return "", err
		} else {
			e.Click()
		}

		if e, err := driver.FindElement(selenium.ByID, "next-button"); err != nil {
			return "", err
		} else {
			e.Click()
		}

		if e, err := driver.FindElement(selenium.ByID, "done-button"); err != nil {
			return "", err
		} else {
			e.Click()
		}
	}

	timeout := time.NewTimer(time.Hour)
	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-timeout.C:
			return "", errors.New("upload timeout")
		default:
			if e, err := driver.FindElement(selenium.ByCSSSelector, "a.style-scope.ytcp-video-info"); err != nil {
				<-ticker.C
			} else {
				href, err := e.GetAttribute("href")
				if err != nil {
					return "", err
				}
				if href == "" {
					<-ticker.C
				} else {
					return href, nil
				}
			}
		}

	}
}

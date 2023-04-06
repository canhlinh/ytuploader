package ytuploader

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
)

var DefaultChromedriverPort = 4444
var DefaultUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36"
var DefaultBrowserCloseDuration = 5 * time.Second

// YtUploader presents an uploader
type YtUploader struct {
	scrPath              string
	browserCloseDuration time.Duration
}

// New creates a new upload instance
func New(screenshotPath string) *YtUploader {

	return &YtUploader{
		scrPath:              screenshotPath,
		browserCloseDuration: DefaultBrowserCloseDuration,
	}
}

// Upload uploads file to Youtube
func (ul *YtUploader) Upload(channel string, filename string, cookies []*http.Cookie, save bool) (string, error) {
	service, err := selenium.NewChromeDriverService("chromedriver", DefaultChromedriverPort)
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
		"--headless", // comment out this line to see the browser
		"--user-agent=" + DefaultUserAgent,
	}})

	driver, err := selenium.NewRemote(caps, "")
	if err != nil {
		return "", err
	}
	defer driver.Close()

	if err := driver.Get("https://www.youtube.com/?persist_gl=1&gl=US&persist_hl=1&hl=en"); err != nil {
		return "", err
	}

	for _, cookie := range cookies {
		if err := driver.AddCookie(&selenium.Cookie{
			Name:   cookie.Name,
			Value:  cookie.Value,
			Path:   cookie.Path,
			Domain: cookie.Domain,
			Secure: cookie.Secure,
			Expiry: uint(cookie.Expires.Unix()),
		}); err != nil {
			return "", err
		}
	}

	uploadURL := "https://youtube.com/upload?persist_gl=1&gl=US&persist_hl=1&hl=en"
	uploadToChannel := false

	if channel != "" {
		uploadURL = fmt.Sprintf("https://studio.youtube.com/channel/%s?persist_gl=1&gl=US&persist_hl=1&hl=en", channel)
		uploadToChannel = true
	}

	if err := driver.Get(uploadURL); err != nil {
		return "", err
	}

	time.Sleep(time.Second)

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
		_, err := wd.FindElement(selenium.ByCSSSelector, ".error-area.style-scope.ytcp-uploads-dialog")
		return err == nil, nil
	}, 3*time.Second); err != nil {
		return "", errors.New("failed to get ytcp-uploads-dialog. timeout")
	}

	bar := progressbar.NewOptions(100,
		progressbar.OptionShowBytes(false),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetDescription("Uploading..."),
	)
	if err := driver.WaitWithTimeout(func(wd selenium.WebDriver) (bool, error) {
		curProgress := currentUploadProgress(wd)
		bar.Set(curProgress)
		return curProgress == 100, nil
	}, 1*time.Hour); err != nil {
		return "", errors.New("failed to upload video. timeout")
	}
	bar.Finish()
	bar.Close()

	url, err := getVideoURL(driver)
	if err != nil {
		return "", err
	}
	if save {
		driver.ExecuteScript(`document.getElementById('toggle-button').scrollIntoView(false);`, nil)

		if e, err := driver.FindElement(selenium.ByName, "VIDEO_MADE_FOR_KIDS_NOT_MFK"); err != nil {
			return "", err
		} else {
			e.Click()
		}

		time.Sleep(1 * time.Second)
		if e, err := driver.FindElement(selenium.ByID, "next-button"); err != nil {
			return "", err
		} else {
			e.Click()
		}

		time.Sleep(1 * time.Second)
		if e, err := driver.FindElement(selenium.ByID, "next-button"); err != nil {
			return "", err
		} else {
			e.Click()
		}

		time.Sleep(1 * time.Second)

		if e, err := driver.FindElement(selenium.ByID, "next-button"); err != nil {
			return "", err
		} else {
			e.Click()
		}

		time.Sleep(1 * time.Second)
		if e, err := driver.FindElement(selenium.ByID, "done-button"); err != nil {
			return "", err
		} else {
			e.Click()
		}

		time.Sleep(3 * time.Second)
	} else {
		time.Sleep(ul.browserCloseDuration)
	}

	if len(ul.scrPath) > 0 {
		if data, err := driver.Screenshot(); err == nil {
			ioutil.WriteFile(ul.scrPath, data, 0644)
		}
	}
	return url, err
}

func currentUploadProgress(wd selenium.WebDriver) int {
	if e, err := wd.FindElement(selenium.ByXPATH, `//tp-yt-paper-progress[contains(@class,"ytcp-video-upload-progress-hover")]`); err == nil {
		rawValue, _ := e.GetAttribute("value")
		value, _ := strconv.Atoi(rawValue)
		return value
	}
	return 0
}

func getVideoURL(wd selenium.WebDriver) (string, error) {
	bar := progressbar.NewOptions(-1,
		progressbar.OptionSetWidth(15),
		progressbar.OptionSpinnerType(9),
		progressbar.OptionSetDescription("Generating video url"),
	)

	defer func() {
		bar.Close()
		fmt.Println()
	}()

	timeout := time.NewTimer(3 * time.Minute)
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-timeout.C:
			return "", errors.New("upload timeout")
		default:
			if e, err := wd.FindElement(selenium.ByCSSSelector, "a.style-scope.ytcp-video-info"); err != nil {
				<-ticker.C
			} else {
				href, err := e.GetAttribute("href")
				if err != nil {
					return "", err
				}
				if href == "" {
					bar.Add(1)
					<-ticker.C
				} else {
					return href, nil
				}
			}
		}

	}
}

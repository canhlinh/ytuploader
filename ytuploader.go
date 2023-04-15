package ytuploader

import (
	"errors"
	"fmt"
	"io/ioutil"
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
var DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36"
var DefaultBrowserCloseDuration = 5 * time.Second

const (
	YoutuybeUploadURL  = "https://youtube.com/upload?persist_gl=1&gl=US&persist_hl=1&hl=en"
	YoutubeHomepageURL = "https://www.youtube.com/?persist_gl=1&gl=US&persist_hl=1&hl=en"
)

// YtUploader presents an uploader
type YtUploader struct {
	screenshotFolder     string
	browserCloseDuration time.Duration
	account              string
}

// New creates a new upload instance
func New(screenshotFolder string, account string) *YtUploader {

	return &YtUploader{
		screenshotFolder:     screenshotFolder,
		browserCloseDuration: DefaultBrowserCloseDuration,
		account:              account,
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
		"--user-agent=", DefaultUserAgent,
		"--profile-directory=", ul.account,
	}})

	driver, err := selenium.NewRemote(caps, "http://127.0.0.1:4444/wd/hub")
	if err != nil {
		return "", err
	}
	defer driver.Close()

	if err := driver.Get(YoutubeHomepageURL); err != nil {
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
	time.Sleep(time.Second * 1)
	if err := driver.Get(YoutubeHomepageURL); err != nil {
		return "", err
	}
	time.Sleep(time.Second * 3)
	if err := driver.Get(YoutuybeUploadURL); err != nil {
		return "", err
	}

	time.Sleep(time.Second * 3)
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

	ul.takeScreenshoot(driver, filename)
	return url, err
}

func (ul *YtUploader) takeScreenshoot(driver selenium.WebDriver, filename string) {
	if data, err := driver.Screenshot(); err == nil {
		ioutil.WriteFile(filepath.Join(ul.screenshotFolder, ul.account, filename), data, 0644)
	}
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

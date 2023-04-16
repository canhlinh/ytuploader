package ytuploader

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/schollz/progressbar/v3"
)

var DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36"
var DefaultBrowserCloseDuration = 1 * time.Second

const (
	YoutuybeUploadURL    = "https://youtube.com/upload?persist_gl=1&gl=US&persist_hl=1&hl=en"
	YoutubeHomepageURL   = "https://www.youtube.com/?persist_gl=1&gl=US&persist_hl=1&hl=en"
	BypassHeadlessScript = `(function(w, n, wn) {
		// Pass the Webdriver Test.
		Object.defineProperty(n, 'webdriver', {
		  get: () => false,
		});
	  
		// Pass the Plugins Length Test.
		// Overwrite the plugins property to use a custom getter.
		Object.defineProperty(n, 'plugins', {
		  // This just needs to have length > 0 for the current test,
		  // but we could mock the plugins too if necessary.
		  get: () => [1, 2, 3, 4, 5],
		});
	  
		// Pass the Languages Test.
		// Overwrite the plugins property to use a custom getter.
		Object.defineProperty(n, 'languages', {
		  get: () => ['en-US', 'en'],
		});
	  
		// Pass the Chrome Test.
		// We can mock this in as much depth as we need for the test.
		w.chrome = {
		  runtime: {},
		};
	  
		// Pass the Permissions Test.
		const originalQuery = wn.permissions.query;
		return wn.permissions.query = (parameters) => (
		  parameters.name === 'notifications' ?
			Promise.resolve({ state: Notification.permission }) :
			originalQuery(parameters)
		);
	  
	  })(window, navigator, window.navigator);`
)

// YtUploader presents an uploader
type YtUploader struct {
	screenshotFolder string
	account          string
	userAgent        string
	ctx              context.Context
	ctxCancel        context.CancelFunc
	Headless         bool
}

// New creates a new upload instance
func New(screenshotFolder string, account string, userAgent string) *YtUploader {
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}

	uploader := &YtUploader{
		screenshotFolder: screenshotFolder,
		account:          account,
		userAgent:        userAgent,
		Headless:         true,
	}
	return uploader
}

func (u *YtUploader) startBrowser() error {
	log.Println("starting browser")

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("start-fullscreen", true),
		chromedp.Flag("enable-automation", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("profile-directory", u.account),
		chromedp.Flag("user-agent", u.userAgent),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("headless", u.Headless),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("mute-audio", true),
	)
	ctx, _ := chromedp.NewExecAllocator(context.Background(), opts...)
	u.ctx, u.ctxCancel = chromedp.NewContext(ctx)
	return chromedp.Run(u.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		if _, err := page.AddScriptToEvaluateOnNewDocument(BypassHeadlessScript).Do(ctx); err != nil {
			return err
		}
		return nil
	}))
}

func (u *YtUploader) closeBrowser() {
	log.Println("closing browser")
	u.ctxCancel()
}

func (u *YtUploader) Upload(channel string, filename string, cookies []*http.Cookie, save bool) (string, error) {
	if err := u.startBrowser(); err != nil {
		return "", err
	}
	defer u.closeBrowser()

	videoURL, err := u.upload(channel, filename, cookies, save)
	if err != nil {
		u.capture("error_at:%s" + time.Now().Format(time.RFC3339))
	}
	return videoURL, err
}

// Upload uploads file to Youtube
func (u *YtUploader) upload(channel string, filename string, cookies []*http.Cookie, save bool) (string, error) {
	if err := u.setCookies(YoutubeHomepageURL, cookies...); err != nil {
		return "", fmt.Errorf("failed to get cookies %s", err.Error())
	}

	if err := u.submitFile(filename); err != nil {
		return "", fmt.Errorf("failed to submit file %s", err.Error())
	}

	if err := u.waitingUploadCompleted(); err != nil {
		return "", fmt.Errorf("failed to waiting upload %s", err.Error())
	}

	videoURL, err := u.getVideoURL()
	if err != nil {
		return "", fmt.Errorf("failed to get videoURL %s", err.Error())
	}
	if save {
		if err := u.saveVideo(); err != nil {
			return "", fmt.Errorf("failed to save video %s", err.Error())
		}
	}

	time.Sleep(time.Second)
	u.capture(filename)
	return videoURL, nil
}

// capture takes a fullscreen shoot
func (u *YtUploader) capture(filename string) {
	log.Println("taking screenshot")

	folder := filepath.Join(u.screenshotFolder, u.account)
	if err := os.MkdirAll(folder, os.ModePerm); err != nil {
		log.Println("error:" + err.Error())
	}
	filePath := filepath.Join(folder, filename+".jpg")

	var buf []byte
	if err := chromedp.Run(u.ctx, chromedp.FullScreenshot(&buf, 90)); err != nil {
		return
	}
	file, err := os.Create(filePath)
	if err != nil {
		log.Println(err)
	}
	file.Write(buf)
	file.Close()
}

func (u *YtUploader) saveVideo() error {
	log.Println("saving the video")

	return chromedp.Run(u.ctx,
		chromedp.Evaluate("document.getElementById('toggle-button').scrollIntoView(false);", nil),
		chromedp.Click(`[name="VIDEO_MADE_FOR_KIDS_NOT_MFK"]`, chromedp.ByQuery, chromedp.NodeVisible),
		chromedp.Click("#next-button", chromedp.ByID, chromedp.NodeVisible),
		chromedp.Click("#next-button", chromedp.ByID, chromedp.NodeVisible),
		chromedp.Click("#done-button", chromedp.ByID, chromedp.NodeVisible),
	)
}

func (u *YtUploader) submitFile(filename string) error {
	log.Println("uploading the video")

	absFilePath, err := filepath.Abs(filename)
	if err != nil {
		return err
	}
	if _, err := os.Stat(absFilePath); err != nil {
		return err
	}

	return chromedp.Run(u.ctx, chromedp.Tasks{
		chromedp.Navigate(YoutuybeUploadURL),
		chromedp.WaitVisible("#select-files-button", chromedp.ByID),
		chromedp.SetUploadFiles(`#content > input[type=file]`, []string{absFilePath}),
	})
}

func (u *YtUploader) setCookies(host string, cookies ...*http.Cookie) error {
	log.Println("set cookies")
	timeout, cancel := context.WithTimeout(u.ctx, time.Second*10)
	defer cancel()

	return chromedp.Run(timeout, chromedp.Tasks{
		chromedp.Navigate(host),
		chromedp.ActionFunc(func(ctx context.Context) error {
			for _, cookie := range cookies {
				exp := (cdp.TimeSinceEpoch)(cookie.Expires)
				sameSite := network.CookieSameSite("unspecified")
				switch cookie.SameSite {
				case http.SameSiteLaxMode:
					sameSite = network.CookieSameSiteLax
				case http.SameSiteStrictMode:
					sameSite = network.CookieSameSiteStrict
				case http.SameSiteNoneMode:
					sameSite = network.CookieSameSiteNone
				}
				err := network.SetCookie(cookie.Name, cookie.Value).
					WithExpires(&exp).
					WithDomain(cookie.Domain).
					WithHTTPOnly(cookie.HttpOnly).
					WithSameSite(sameSite).
					WithSecure(cookie.Secure).
					WithPath(cookie.Path).
					Do(ctx)
				if err != nil {
					return fmt.Errorf(cookie.Name)
				}
			}
			return nil
		}),
		chromedp.Navigate(host),
		chromedp.WaitVisible(`#avatar-btn`, chromedp.ByID),
	})
}

func parsePercentage(s string) (int, error) {
	re := regexp.MustCompile(`\d*%`)
	match := re.FindStringSubmatch(s)
	if len(match) == 0 {
		return 0, errors.New("not found")
	} else {
		return strconv.Atoi(strings.TrimSuffix(match[0], "%"))
	}
}

func (u *YtUploader) waitingUploadCompleted() error {
	log.Println("wait uploading complete")

	bar := progressbar.NewOptions(100,
		progressbar.OptionShowBytes(false),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetDescription("Uploading..."),
	)

	for {
		var res string
		if err := chromedp.Run(u.ctx,
			chromedp.Text("#dialog > div > ytcp-animatable.button-area.metadata-fade-in-section.style-scope.ytcp-uploads-dialog > div > div.left-button-area.style-scope.ytcp-uploads-dialog > ytcp-ve > div.error-short.style-scope.ytcp-uploads-dialog", &res)); err == nil {
			if len(res) > 0 {
				return errors.New(res)
			}
		}

		if err := chromedp.Run(u.ctx,
			chromedp.Text(`#dialog > div > ytcp-animatable.button-area.metadata-fade-in-section.style-scope.ytcp-uploads-dialog > div > div.left-button-area.style-scope.ytcp-uploads-dialog > ytcp-video-upload-progress > span`, &res, chromedp.NodeVisible)); err != nil {
			return err
		}
		if strings.Contains(res, "Upload complete") || strings.Contains(res, "Processing up to") {
			bar.Set(100)
			bar.Finish()
			bar.Close()
			break
		} else if strings.Contains(res, "Uploading") {
			progress, _ := parsePercentage(res)
			bar.Set(progress)
		} else {
			return errors.New("something went wrong")
		}
		<-time.After(time.Millisecond * 100)
	}
	log.Println("upload finished")
	return nil
}

func (u *YtUploader) getVideoURL() (string, error) {
	log.Println("getting video url")

	bar := progressbar.NewOptions(-1,
		progressbar.OptionSetWidth(15),
		progressbar.OptionSpinnerType(9),
		progressbar.OptionSetDescription("Generating video url"),
	)

	defer func() {
		bar.Close()
		fmt.Println()
	}()

	timeout := time.NewTimer(1 * time.Minute)
	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-timeout.C:
			return "", errors.New("parse link timeout")
		default:
			var nodes []*cdp.Node
			if err := chromedp.Run(u.ctx, chromedp.Nodes(`a.style-scope.ytcp-video-info`, &nodes, chromedp.NodeVisible)); err != nil {
				return "", err
			}
			href := nodes[0].AttributeValue("href")
			if href != "" {
				return href, nil
			}
			bar.Add(1)
			<-ticker.C
		}
	}
}

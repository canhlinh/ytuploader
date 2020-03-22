package ytuploader

import (
	"encoding/json"
	"net/http"
	"os"
	"time"
)

type Cookie struct {
	Domain         string  `json:"domain"`
	ExpirationDate float64 `json:"expirationDate"`
	HostOnly       bool    `json:"hostOnly"`
	HTTPOnly       bool    `json:"httpOnly"`
	Name           string  `json:"name"`
	Path           string  `json:"path"`
	SameSite       string  `json:"sameSite"`
	Secure         bool    `json:"secure"`
	Session        bool    `json:"session"`
	StoreID        string  `json:"storeId"`
	Value          string  `json:"value"`
	ID             int     `json:"id"`
}

func (c *Cookie) Builtin() *http.Cookie {
	return &http.Cookie{
		Name:     c.Name,
		Domain:   c.Domain,
		Value:    c.Value,
		Path:     c.Path,
		SameSite: http.SameSiteNoneMode,
		Expires:  time.Unix(int64(c.ExpirationDate), 0),
		Secure:   c.Secure,
	}
}

func ParseCookiesFromJSONFile(path string) (Cookies, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	var cookies []*Cookie
	if err := json.NewDecoder(file).Decode(&cookies); err != nil {
		return nil, err
	}

	return cookies, nil
}

type Cookies []*Cookie

func (cookies Cookies) Builtin() []*http.Cookie {
	c := []*http.Cookie{}
	for _, cookie := range cookies {
		c = append(c, cookie.Builtin())
	}
	return c
}

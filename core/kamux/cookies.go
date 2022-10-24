package kamux

import (
	"net/http"
	"strings"
	"time"

	"github.com/kamalshkeir/kago/core/settings"
)

var (
	COOKIES_Expires = 20*time.Second
	COOKIES_SameSite = http.SameSiteStrictMode
	COOKIES_HttpOnly = true
	COOKIES_Secure = false
)

func init() {
	if strings.Contains(settings.Config.Port,"443") {
		COOKIES_Secure=true
	}
}


// SetCookie set cookie given key and value
func (c *Context) SetCookie(key, value string) {
	if !COOKIES_Secure {
		if c.Request.TLS != nil {
			COOKIES_Secure=true
		}
	}
	http.SetCookie(c.ResponseWriter, &http.Cookie{
		Name:     key,
		Value:    value,
		Path:     "/",
		Expires:  time.Now().Add(COOKIES_Expires),
		HttpOnly: COOKIES_HttpOnly,
		SameSite: COOKIES_SameSite,
		Secure: COOKIES_Secure,
		MaxAge: int(COOKIES_Expires.Seconds()),
	})
}

// GetCookie get cookie with specific key
func (c *Context) GetCookie(key string) (string, error) {
	v, err := c.Request.Cookie(key)
	if err != nil {
		return "", err
	}
	return v.Value, nil
}

// DeleteCookie delete cookie with specific key
func (c *Context) DeleteCookie(key string) {
	http.SetCookie(c.ResponseWriter, &http.Cookie{
		Name:     key,
		Value:    "",
		Path:     "/",
		Expires:  time.Now(),
		HttpOnly: COOKIES_HttpOnly,
		SameSite: COOKIES_SameSite,
		Secure: COOKIES_Secure,
		MaxAge: -1,
	})
}

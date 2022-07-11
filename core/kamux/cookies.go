package kamux

import (
	"net/http"
	"time"
)

// COOKIE_EXPIRE global cookie expiry time
var COOKIE_EXPIRE= time.Now().Add(7 * 24 * time.Hour)

// SetCookie set cookie given key and value
func (c *Context) SetCookie(key,value string) {
	http.SetCookie(c.ResponseWriter, &http.Cookie{
		Name: key,
		Value: value,
		Path: "/",
		Expires:COOKIE_EXPIRE,
		HttpOnly: true,
	})
}

// GetCookie get cookie with specific key
func (c *Context) GetCookie(key string) (string,error) {
	v,err := c.Request.Cookie(key)
	if err != nil {
		return "",err
	}
	return v.Value,nil
}

// DeleteCookie delete cookie with specific key
func (c *Context) DeleteCookie(key string) {
	http.SetCookie(c.ResponseWriter, &http.Cookie{
		Name: key,
		Value: "",
		Path: "/",
		Expires: time.Now(),
		HttpOnly: true,
	})
}


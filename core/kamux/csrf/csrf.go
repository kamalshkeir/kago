package csrf

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/kamalshkeir/kago/core/utils"
	"github.com/kamalshkeir/kago/core/utils/encryption/encryptor"
	"github.com/kamalshkeir/kago/core/utils/eventbus"
	"github.com/kamalshkeir/kago/core/utils/safemap"
)

var Used bool
var Csrf_rand = utils.GenerateRandomString(10)
var CSRF_CLEAN_EVERY= 20*time.Minute
var CSRF_TIMEOUT_RETRY = 4
var Csrf_tokens = safemap.New[string,Token]()
var onc sync.Once

type Token struct {
	Used  bool
	Retry int
	Value string
	Remote string
	Created time.Time
}

var CSRF = func(handler http.Handler) http.Handler {
	onc.Do(func() {
		if !Used {
			i := time.Now()
			eventbus.Subscribe("csrf-clean",func(data string) {
				if data != "" {
					Csrf_tokens.Delete(data)
				}
				if time.Since(i) > time.Hour {
					Csrf_tokens.Flush()
					i=time.Now()
				}
			})
			Used=true
		}
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {		
		switch r.Method {
		case "GET":
			token := r.Header.Get("X-CSRF-Token")
			tok,ok := Csrf_tokens.Get(token)

			if token == "" || !ok || token != tok.Value || tok.Retry > CSRF_TIMEOUT_RETRY {
				t,_ := encryptor.Encrypt(Csrf_rand)
				Csrf_tokens.Set(t,Token{
					Value: t,
					Used: false,
					Retry: 0,
					Remote: r.UserAgent(),
					Created: time.Now(),
				})
				http.SetCookie(w, &http.Cookie{
					Name:     "csrf_token",
					Value:    t,
					Path:     "/",
					Expires:  time.Now().Add(5*time.Minute),
					SameSite: http.SameSiteStrictMode,
				})
			} else {
				if token != tok.Value {
					http.SetCookie(w, &http.Cookie{
						Name:     "csrf_token",
						Value:    tok.Value,
						Path:     "/",
						Expires:  time.Now().Add(5*time.Minute),
						SameSite: http.SameSiteStrictMode,
					})
				}
			}

			handler.ServeHTTP(w, r)
			return	
			 
		case "POST","PATCH","PUT","UPDATE","DELETE":
			token := r.Header.Get("X-CSRF-Token")			
			tok,ok := Csrf_tokens.Get(token)
			if !ok || token == "" || tok.Used || tok.Retry > CSRF_TIMEOUT_RETRY || time.Since(tok.Created) > CSRF_CLEAN_EVERY  {
				eventbus.Publish("csrf-clean",tok.Value)
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]any{
					"error": "CSRF not allowed",
				})
				return		
			}
			
		
			Csrf_tokens.Set(tok.Value,Token{
				Value: tok.Value,
				Used: true,
				Retry: tok.Retry+1,
				Remote: r.UserAgent(),
				Created: tok.Created,
			})
			handler.ServeHTTP(w, r)
			return
		default:
			handler.ServeHTTP(w, r)
		}
	})
}





package kamux

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/kamalshkeir/kago/core/admin/models"
	"github.com/kamalshkeir/kago/core/kamux/csrf"
	"github.com/kamalshkeir/kago/core/kamux/gzip"
	"github.com/kamalshkeir/kago/core/kamux/logs"
	"github.com/kamalshkeir/kago/core/kamux/ratelimiter"
	"github.com/kamalshkeir/kago/core/orm"
	"github.com/kamalshkeir/kago/core/utils"
	"github.com/kamalshkeir/kago/core/utils/encryption/encryptor"
	"github.com/kamalshkeir/kago/core/utils/logger"
)



var SESSION_ENCRYPTION = true

// AuthMiddleware can be added to any handler to get user cookie authentication and pass it to handler and templates
var Auth = func(handler Handler) Handler {
	const key utils.ContextKey = "user"
	return func(c *Context) {
		session, err := c.GetCookie("session")
		if err != nil || session == "" {
			// NOT AUTHENTICATED
			c.DeleteCookie("session")
			handler(c)
			return
		}
		if SESSION_ENCRYPTION {
			session, err = encryptor.Decrypt(session)
			if err != nil {
				handler(c)
				return
			}
		}
		// Check session
		user, err := orm.Model[models.User]().Where("uuid = ?", session).One()
		if err != nil {
			// session fail
			handler(c)
			return
		}

		// AUTHENTICATED AND FOUND IN DB
		ctx := context.WithValue(c.Request.Context(), key, user)
		*c = Context{
			ResponseWriter: c.ResponseWriter,
			Request:        c.Request.WithContext(ctx),
			Params:         c.Params,
		}
		handler(c)
	}
}

var Admin = func(handler Handler) Handler {
	const key utils.ContextKey = "user"
	return func(c *Context) {
		session, err := c.GetCookie("session")
		if err != nil || session == "" {
			// NOT AUTHENTICATED
			c.DeleteCookie("session")
			http.Redirect(c.ResponseWriter, c.Request, "/admin/login", http.StatusTemporaryRedirect)
			return
		}
		if SESSION_ENCRYPTION {
			session, err = encryptor.Decrypt(session)
			if err != nil {
				c.Status(http.StatusTemporaryRedirect).Redirect("/admin/login")
				return
			}
		}
		user, err := orm.Model[models.User]().Where("uuid = ?", session).One()

		if err != nil {
			// AUTHENTICATED BUT NOT FOUND IN DB
			c.Status(http.StatusTemporaryRedirect).Redirect("/admin/login")
			return
		}

		// Not admin
		if !user.IsAdmin {
			c.Status(403).Text("Middleware : Not allowed to access this page")
			return
		}

		ctx := context.WithValue(c.Request.Context(), key, user)
		*c = Context{
			ResponseWriter: c.ResponseWriter,
			Request:        c.Request.WithContext(ctx),
			Params:         c.Params,
		}

		handler(c)
	}
}

var BasicAuth = func(next Handler, user, pass string) Handler {
	return func(c *Context) {
		// Extract the username and password from the request
		// Authorization header. If no Authentication header is present
		// or the header value is invalid, then the 'ok' return value
		// will be false.
		username, password, ok := c.Request.BasicAuth()
		if ok {
			// Calculate SHA-256 hashes for the provided and expected
			// usernames and passwords.
			usernameHash := sha256.Sum256([]byte(username))
			passwordHash := sha256.Sum256([]byte(password))
			expectedUsernameHash := sha256.Sum256([]byte(user))
			expectedPasswordHash := sha256.Sum256([]byte(pass))

			// Use the subtle.ConstantTimeCompare() function to check if
			// the provided username and password hashes equal the
			// expected username and password hashes. ConstantTimeCompare
			// will return 1 if the values are equal, or 0 otherwise.
			// Importantly, we should to do the work to evaluate both the
			// username and password before checking the return values to
			// avoid leaking information.
			usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
			passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)

			// If the username and password are correct, then call
			// the next handler in the chain. Make sure to return
			// afterwards, so that none of the code below is run.
			if usernameMatch && passwordMatch {
				next(c)
				return
			}
		}

		// If the Authentication header is not present, is invalid, or the
		// username or password is wrong, then set a WWW-Authenticate
		// header to inform the client that we expect them to use basic
		// authentication and send a 401 Unauthorized response.
		c.ResponseWriter.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(c.ResponseWriter, "Unauthorized", http.StatusUnauthorized)
	}
}

var Csrf = func(handler Handler) Handler {
	return func(c *Context) {
		switch c.Method {
		case "GET":
			token := c.Request.Header.Get("X-CSRF-Token")
			tok,ok := csrf.Csrf_tokens.Get(token)

			if token == "" || !ok || token != tok.Value || tok.Retry > csrf.CSRF_TIMEOUT_RETRY{
				t,_ := encryptor.Encrypt(csrf.Csrf_rand)
				csrf.Csrf_tokens.Set(t,csrf.Token{
					Value: t,
					Used: false,
					Retry: 0,
					Remote: c.Request.UserAgent(),
					Created: time.Now(),
				})
				http.SetCookie(c.ResponseWriter, &http.Cookie{
					Name:     "csrf_token",
					Value:    t,
					Path:     "/",
					Expires:  time.Now().Add(csrf.CSRF_CLEAN_EVERY),
					SameSite: http.SameSiteStrictMode,
				})
			} else {
				if token != tok.Value {
					http.SetCookie(c.ResponseWriter, &http.Cookie{
						Name:     "csrf_token",
						Value:    tok.Value,
						Path:     "/",
						Expires:  time.Now().Add(csrf.CSRF_CLEAN_EVERY),
						SameSite: http.SameSiteStrictMode,
					})
				}
			}
		case "POST","PATCH","PUT","UPDATE","DELETE":
			token := c.Request.Header.Get("X-CSRF-Token")			
			tok,ok := csrf.Csrf_tokens.Get(token)
			if !ok || token == "" || tok.Used || tok.Retry > csrf.CSRF_TIMEOUT_RETRY || time.Since(tok.Created) > csrf.CSRF_CLEAN_EVERY || c.Request.UserAgent() != tok.Remote {
				c.ResponseWriter.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(c.ResponseWriter).Encode(map[string]any{
					"error": "CSRF not allowed",
				})
				return	
				
			}
			csrf.Csrf_tokens.Set(tok.Value,csrf.Token{
				Value: tok.Value,
				Used: true,
				Retry: tok.Retry+1,
				Remote: c.Request.UserAgent(),
				Created: tok.Created,
			})
		}
		handler(c)
	}
}





var corsAdded = false
var Origines = []string{}
func (router *Router) AllowOrigines(origines ...string) {
	if !corsAdded {
		midwrs = append(midwrs, cors)
		corsAdded=true
	}
	Origines = append(Origines, origines...)
}

var cors = func(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set headers
		o := strings.Join(Origines,",")
		w.Header().Set("Access-Control-Allow-Origin", o)
		w.Header().Set("Access-Control-Allow-Headers:", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		// Next
		next.ServeHTTP(w, r)
	})
}


var RECOVERY = func(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err != nil {
				logger.Error(err) // May be log this error?
				jsonBody, _ := json.Marshal(map[string]string{
					"error": "There was an internal server error",
				})
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(jsonBody)
			}
		}()
		next.ServeHTTP(w, r)
	})
}




var CSRF=csrf.CSRF
var GZIP = gzip.GZIP
var LIMITER = ratelimiter.LIMITER
var LOGS = logs.LOGS





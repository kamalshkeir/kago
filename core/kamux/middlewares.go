package kamux

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kamalshkeir/kago/core/admin/models"
	"github.com/kamalshkeir/kago/core/kamux/csrf"
	"github.com/kamalshkeir/kago/core/kamux/gzip"
	"github.com/kamalshkeir/kago/core/kamux/logs"
	"github.com/kamalshkeir/kago/core/orm"
	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils"
	"github.com/kamalshkeir/kago/core/utils/encryption/encryptor"
	"github.com/kamalshkeir/kago/core/utils/eventbus"
	"github.com/kamalshkeir/kago/core/utils/logger"

	"golang.org/x/time/rate"
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

var CSRF = func(handler http.Handler) http.Handler {
	// generate token
	tokBytes := make([]byte, 64)
	_, err := io.ReadFull(rand.Reader, tokBytes)
	logger.CheckError(err)

	massToken := csrf.MaskToken(tokBytes)
	toSendToken := base64.StdEncoding.EncodeToString(massToken)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			token := r.Header.Get("X-CSRF-Token")
			if token == "" {
				http.SetCookie(w, &http.Cookie{
					Name:     "csrf_token",
					Value:    toSendToken,
					Path:     "/",
					Expires:  time.Now().Add(1 * time.Hour),
					Secure:   true,
					SameSite: http.SameSiteStrictMode,
				})
			}
		} else if r.Method == "POST" {
			token := r.Header.Get("X-CSRF-Token")
			if !csrf.VerifyToken(token, toSendToken) {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": "CSRF not allowed",
				})
				return
			}
		}
		handler.ServeHTTP(w, r)
	})
}

var CORS = func(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set headers
		w.Header().Set("Access-Control-Allow-Headers:", "*")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		// Next
		next.ServeHTTP(w, r)
	})
}

var GZIP = func(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "metrics") {
			handler.ServeHTTP(w, r)
			return
		}
		//check if connection is ws
		for _, header := range r.Header["Upgrade"] {
			if header == "websocket" {
				// connection is ws
				handler.ServeHTTP(w, r)
				return
			}
		}
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			gwriter := gzip.NewWrappedResponseWriter(w)
			defer gwriter.Flush()
			gwriter.Header().Set("Content-Encoding", "gzip")
			handler.ServeHTTP(gwriter, r)
			return
		}
		handler.ServeHTTP(w, r)
	})
}

var banned = sync.Map{}
var LIMITER_TOKENS = 50
var LIMITER_TIMEOUT = 5 * time.Minute
var LIMITER = func(next http.Handler) http.Handler {
	var limiter = rate.NewLimiter(1, LIMITER_TOKENS)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v, ok := banned.Load(r.RemoteAddr)
		if ok {
			if time.Since(v.(time.Time)) > LIMITER_TIMEOUT {
				banned.Delete(r.RemoteAddr)
			} else {
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("<h1>YOU DID TOO MANY REQUEST, YOU HAVE BEEN BANNED FOR 5 MINUTES </h1>"))
				banned.Store(r.RemoteAddr, time.Now())
				return
			}
		}
		if !limiter.Allow() {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("<h1>YOU DID TOO MANY REQUEST, YOU HAVE BEEN BANNED FOR 5 MINUTES </h1>"))
			banned.Store(r.RemoteAddr, time.Now())
			return
		}
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

var LOGS = func(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if utils.StringContains(r.URL.Path, "metrics", "sw.js", "favicon", "/static/", "/sse/", "/ws/", "/wss/") {
			h.ServeHTTP(w, r)
			return
		}
		//check if connection is ws
		for _, header := range r.Header["Upgrade"] {
			if header == "websocket" {
				// connection is ws
				h.ServeHTTP(w, r)
				return
			}
		}
		recorder := &logs.StatusRecorder{
			ResponseWriter: w,
			Status:         200,
		}
		t := time.Now()
		h.ServeHTTP(recorder, r)
		res := fmt.Sprintf("[%s] --> '%s' --> [%d]  from: %s ---------- Took: %v", r.Method, r.URL.Path, recorder.Status, r.RemoteAddr, time.Since(t))

		if recorder.Status >= 200 && recorder.Status < 400 {
			fmt.Printf(logger.Green, res)
		} else if recorder.Status >= 400 || recorder.Status < 200 {
			fmt.Printf(logger.Red, res)
		} else {
			fmt.Printf(logger.Yellow, res)
		}
		if settings.Config.Logs {
			logger.StreamLogs = append(logger.StreamLogs, res)
			eventbus.Publish("internal-logs", map[string]string{})
		}
	})
}



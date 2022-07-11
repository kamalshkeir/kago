package middlewares

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kamalshkeir/kago/core/kamux"
	"github.com/kamalshkeir/kago/core/middlewares/csrf"
	"github.com/kamalshkeir/kago/core/middlewares/gzip"
	"github.com/kamalshkeir/kago/core/orm"
	"github.com/kamalshkeir/kago/core/utils"
	"github.com/kamalshkeir/kago/core/utils/encryption/encryptor"
	"github.com/kamalshkeir/kago/core/utils/logger"

	"golang.org/x/time/rate"
)

var SESSION_ENCRYPTION = true

// AuthMiddleware can be added to any handler to get user cookie authentication and pass it to handler and templates
func Auth(handler kamux.Handler) kamux.Handler {
	const key utils.ContextKey = "user"
	return func(c *kamux.Context) {
		session,err := c.GetCookie("session")
		if err != nil || session == "" {
			// NOT AUTHENTICATED
			handler(c)
			return
		}
		if SESSION_ENCRYPTION {
			session,err = encryptor.Decrypt(session)
			if err != nil {
				c.DeleteCookie("session")
				handler(c)
				return
			}
		}
		// Check session
		user,err := orm.Database().Table("users").Where("uuid = ?",session).One()
		if err != nil {
			// session fail
			handler(c)
			return
		}

		// AUTHENTICATED AND FOUND IN DB
		ctx := context.WithValue(c.Request.Context(),key,user)
		*c = kamux.Context{
			ResponseWriter: c.ResponseWriter,
			Request: c.Request.WithContext(ctx),
			Params: c.Params,
		}
		handler(c)
	}
}

func Admin(handler kamux.Handler) kamux.Handler {
	const key utils.ContextKey = "user"
	return func(c *kamux.Context) {
		session,err := c.GetCookie("session")
		if err != nil || session == "" {
			// NOT AUTHENTICATED
			http.Redirect(c.ResponseWriter,c.Request,"/admin/login",http.StatusSeeOther)
			return
		}
		if SESSION_ENCRYPTION {
			session,err = encryptor.Decrypt(session)
			if err != nil {
				c.DeleteCookie("session")
				c.Redirect("/admin/login",http.StatusSeeOther)
				return
			}
		}
		user,err := orm.Database().Table("users").Where("uuid = ?",session).One()
		
		if err != nil {
			// AUTHENTICATED BUT NOT FOUND IN DB
			c.Redirect("/admin/login",http.StatusSeeOther)
			return
		}

		// Not admin
		if user["is_admin"] == int64(0) || user["is_admin"] == 0 || user["is_admin"] == false {
			c.Text(403, "Not allowed to access this page")
			return
		}

		ctx := context.WithValue(c.Request.Context(),key,user)
		*c = kamux.Context{
			ResponseWriter: c.ResponseWriter,
			Request: c.Request.WithContext(ctx),
			Params: c.Params,
		}

		handler(c)
	}
}

func CSRF(handler http.Handler) http.Handler {
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
					Name: "csrf_token",
					Value: toSendToken,
					Path: "/",
					Expires:time.Now().Add(1 * time.Hour),
					HttpOnly: true,
					Secure: true,
					SameSite: http.SameSiteStrictMode,
				})
			} 
		} else if r.Method == "POST" {
			token := r.Header.Get("X-CSRF-Token")
			if !csrf.VerifyToken(token, toSendToken) {
				w.WriteHeader(200)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": "CSRF not allowed !",
				})
				return
			}
		}
		handler.ServeHTTP(w,r)
	})
}

func CORS(next http.Handler) http.Handler {
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

func GZIP(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path,"metrics") {
			handler.ServeHTTP(w,r)
			return
		}
		//check if connection is ws
		for _, header := range r.Header["Upgrade"] {
			if header == "websocket" {
				// connection is ws
				handler.ServeHTTP(w,r)
				return
			}
		}
		if strings.Contains(r.Header.Get("Accept-Encoding"),"gzip") {
			gwriter := gzip.NewWrappedResponseWriter(w)
			gwriter.Header().Set("Content-Encoding","gzip")
			handler.ServeHTTP(gwriter,r)
			defer gwriter.Flush()
			return
		}
		handler.ServeHTTP(w,r)
	})
}

var banned = sync.Map{}
func Limiter(next http.Handler) http.Handler {
	var limiter = rate.NewLimiter(1,50)
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v,ok := banned.Load(r.RemoteAddr)
		if ok {
			if time.Since(v.(time.Time)) > time.Hour {
				banned.Delete(r.RemoteAddr)
			} else {
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("<h1>YOU DID TOO MANY REQUEST, YOU HAVE BEEN BANNED FOR 60 MINUTES </h1>"))
				banned.Store(r.RemoteAddr,time.Now())
            	return
			}
		}
        if !limiter.Allow() {
            w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("<h1>YOU DID TOO MANY REQUEST, YOU HAVE BEEN BANNED FOR 60 MINUTES </h1>"))
			banned.Store(r.RemoteAddr,time.Now())
            return
        }
        next.ServeHTTP(w, r)
    })
}



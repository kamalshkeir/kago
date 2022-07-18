package kago

import (
	"github.com/kamalshkeir/kago/core/admin"
	"github.com/kamalshkeir/kago/core/kamux"
)

func New(env_files ...string) *kamux.Router {
	// init server and router
	app := kamux.New()
	// init admin urls
	go admin.UrlPatterns(app)
	return app
}
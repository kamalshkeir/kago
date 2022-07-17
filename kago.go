package main

import (
	"github.com/kamalshkeir/kago/core/admin"
	"github.com/kamalshkeir/kago/core/kamux"
	"github.com/kamalshkeir/kago/core/middlewares"
)

func New(env_files ...string) *kamux.Router {
	// init server and router
	app := kamux.New()
	// init admin urls
	admin.UrlPatterns(app)
	return app
}

func main() {
	app := New()
	app.UseMiddlewares(middlewares.GZIP)
	app.Run()
}
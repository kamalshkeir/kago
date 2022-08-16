package kago

import (
	"github.com/kamalshkeir/kago/core/admin"
	"github.com/kamalshkeir/kago/core/kamux"
)

func New() *kamux.Router {
	app := kamux.New()
	admin.UrlPatterns(app)
	return app
}

func BareBone() *kamux.Router {
	app := kamux.BareBone()
	return app
}

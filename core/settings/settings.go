package settings

import "github.com/kamalshkeir/kago/core/utils/safemap"

var MODE = "default"
var Config = &GlobalConfig{}
var Secret string
var Translations = safemap.New[string, map[string]any]()
var TranslationFolder = "translations"
var REPO_NAME = "kago-assets"
var REPO_USER = "kamalshkeir"
var STATIC_DIR= "assets/static"
var TEMPLATE_DIR= "assets/templates"
var MEDIA_DIR = "media"
var Languages = []string{}

type GlobalConfig struct {
	Host  string `env:"HOST|localhost"`
	Port  string `env:"PORT|9313"`
	Embed struct {
		Static    bool `env:"EMBED_STATIC|false"`
		Templates bool `env:"EMBED_TEMPLATES|false"`
	}
	Db struct {
		Name string `env:"DB_NAME|db"`
		Type string `env:"DB_TYPE|sqlite"`
		DSN  string `env:"DB_DSN|"`
	}
	Smtp struct {
		Email string `env:"SMTP_EMAIL|"`
		Pass  string `env:"SMTP_PASS|"`
		Host  string `env:"SMTP_HOST|"`
		Port  string `env:"SMTP_PORT|"`
	}
	Profiler   bool `env:"PROFILER|false"`
	Docs       bool `env:"DOCS|false"`
	Logs       bool `env:"LOGS|false"`
	Monitoring bool `env:"MONITORING|false"`
	Cert 	   string `env:"CERT|"`
	Key 	   string `env:"KEY|"`
	Domain 	   string `env:"DOMAIN|"`
}

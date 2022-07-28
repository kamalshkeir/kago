package settings

import "github.com/kamalshkeir/kago/core/utils/safemap"

var GlobalConfig = &Config{}
var Secret string
var Translations = safemap.New[string, map[string]any]()
var TranslationFolder = "translations"
var REPO_NAME = "kago-assets"
var REPO_USER = "kamalshkeir"
var Languages = []string{}

type Config struct {
	Host           string `env:"HOST|localhost"`
	Port           string `env:"PORT|9313"`
	Profiler       bool   `env:"PROFILER|false"`
	Docs           bool   `env:"DOCS|false"`
	Logs           bool   `env:"LOGS|false"`
	Monitoring     bool   `env:"MONITORING|false"`
	EmbedStatic    bool   `env:"EMBED_STATIC|false"`
	EmbedTemplates bool   `env:"EMBED_TEMPLATES|false"`
	DbType         string `env:"DB_TYPE|sqlite"`
	DbDSN          string `env:"DB_DSN"`
	DbName         string `env:"DB_NAME|db"`
	SmtpEmail      string `env:"SMTP_EMAIL"`
	SmtpPass       string `env:"SMTP_PASS"`
	SmtpHost       string `env:"SMTP_HOST"`
	SmtpPort       string `env:"SMTP_PORT"`
}

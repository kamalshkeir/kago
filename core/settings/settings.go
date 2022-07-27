package settings

import "github.com/kamalshkeir/kago/core/utils/safemap"

var GlobalConfig = &config{}
var Secret string
var Translations = safemap.New[string, map[string]any]()
var TranslationFolder = "translations"
var REPO_NAME = "kago-assets"
var REPO_USER = "kamalshkeir"
var Languages = []string{}

type config struct {
	Host           string
	Port           string
	Profiler       bool
	Docs           bool
	Logs           bool
	Monitoring     bool
	EmbedStatic    bool
	EmbedTemplates bool
	DbType         string
	DbDSN          string
	DbName         string
	SmtpEmail      string
	SmtpPass       string
	SmtpHost       string
	SmtpPort       string
}
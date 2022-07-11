package kamux

import (
	"embed"
	"os"
	"strconv"

	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils/envloader"
	"github.com/kamalshkeir/kago/core/utils/logger"
)

// LoadEnv load env vars from multiple files
func (router *Router) LoadEnv(files ...string) {
	m,err := envloader.LoadToMap(files...)
	if err != nil {
		return
	}
	for k,v := range m {
		switch k {
		case "SECRET":
			settings.GlobalConfig.Secret=v
		case "EMBED_STATIC":
			if b,err := strconv.ParseBool(v);!logger.CheckError(err) {
				settings.GlobalConfig.EmbedStatic=b
			}
		case "EMBED_TEMPLATES":
			if b,err := strconv.ParseBool(v);!logger.CheckError(err) {
				settings.GlobalConfig.EmbedTemplates=b
			}
		case "DB_TYPE":
			if v == "" {v="sqlite"}
			settings.GlobalConfig.DbType=v
		case "DB_DSN":
			if v == "" {v="db.sqlite"}
			settings.GlobalConfig.DbDSN=v
		case "DB_NAME":
			if v == "" {
				logger.Error("DB_NAME from env file cannot be empty")
				os.Exit(1)
			}
			settings.GlobalConfig.DbName=v
		case "SMTP_EMAIL":
			settings.GlobalConfig.SmtpEmail=v
		case "SMTP_PASS":
			settings.GlobalConfig.SmtpPass=v
		case "SMTP_HOST":
			settings.GlobalConfig.SmtpHost=v
		case "SMTP_PORT":
			settings.GlobalConfig.SmtpPort=v
		}
	}
}


var Templates embed.FS
var Static embed.FS
// GetEmbeded get embeded files and make them global
func (r *Router) Embed(staticDir *embed.FS, templateDir *embed.FS) {
	Static = *staticDir
	Templates = *templateDir
}

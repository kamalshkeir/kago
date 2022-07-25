package kamux

import (
	"embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils/envloader"
	"github.com/kamalshkeir/kago/core/utils/logger"
	"github.com/kamalshkeir/kago/core/utils/safemap"
)


var mCountryLanguage=safemap.New[string,string]()

// LoadEnv load env vars from multiple files
func (router *Router) LoadEnv(files ...string) {
	m,err := envloader.LoadToMap(files...)
	if err != nil {
		return
	}
	for k,v := range m {
		switch k {
		case "SECRET":
			settings.Secret=v
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

func (router *Router) PrintServerStart() {
	host := settings.GlobalConfig.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := settings.GlobalConfig.Port
	if port == "" {
		port = "9313"
	}
	fmt.Printf(logger.Yellow, logger.Ascii7)
	fmt.Printf(logger.Blue, "-------⚡🚀 http://"+host+":"+port+" 🚀⚡-------")
}

func LoadTranslations() {
	if dir,err := os.Stat(settings.TranslationFolder);err == nil && dir.IsDir() {
		err = filepath.WalkDir(dir.Name(),func(path string, d fs.DirEntry, err error) error {
			if strings.HasSuffix(d.Name(),".json") {
				file,err := os.Open(path)
				if err != nil {
					return err
				}

				v := map[string]any{}
				err = json.NewDecoder(file).Decode(&v)
				if err != nil {
					file.Close()
					return err
				}
				file.Close()
				withoutSuffix := strings.TrimSuffix(d.Name(),".json")
				settings.Languages = append(settings.Languages, withoutSuffix)
				settings.Translations.Set(withoutSuffix,v)
			}
			return nil
		})
		if !logger.CheckError(err) {
			var res *http.Response
			res,err = http.Get("https://raw.githubusercontent.com/kamalshkeir/countries/main/country_list.csv")
			logger.CheckError(err)
			defer res.Body.Close()
			reader := csv.NewReader(res.Body)
			reader.LazyQuotes=true
			lines, err := reader.ReadAll()
			logger.CheckError(err)

			for _,l := range lines {
				country := l[1]
				lang := l[5]
				for _,ll := range settings.Languages {
					if lang == ll {
						mCountryLanguage.Set(country,lang)
					} 
				}
			}
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

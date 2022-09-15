package kamux

import (
	"embed"
	"encoding/csv"
	"encoding/json"
	"flag"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils"
	"github.com/kamalshkeir/kago/core/utils/envloader"
	"github.com/kamalshkeir/kago/core/utils/logger"
	"github.com/kamalshkeir/kago/core/utils/safemap"
)

var mCountryLanguage = safemap.New[string, string]()

// LoadEnv load env vars from multiple files
func (router *Router) LoadEnv(files ...string) {
	envloader.Load(files...)
	err := envloader.FillStruct(settings.Config)
	logger.CheckError(err)
}

func LoadTranslations() {
	if dir, err := os.Stat(settings.TranslationFolder); err == nil && dir.IsDir() {
		err = filepath.WalkDir(dir.Name(), func(path string, d fs.DirEntry, err error) error {
			if strings.HasSuffix(d.Name(), ".json") {
				file, err := os.Open(path)
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
				withoutSuffix := strings.TrimSuffix(d.Name(), ".json")
				settings.Languages = append(settings.Languages, withoutSuffix)
				settings.Translations.Set(withoutSuffix, v)
			}
			return nil
		})
		if !logger.CheckError(err) {
			var res *http.Response
			res, err = http.Get("https://raw.githubusercontent.com/kamalshkeir/countries/main/country_list.csv")
			logger.CheckError(err)
			defer res.Body.Close()
			reader := csv.NewReader(res.Body)
			reader.LazyQuotes = true
			lines, err := reader.ReadAll()
			logger.CheckError(err)

			for _, l := range lines {
				country := l[1]
				lang := l[5]
				for _, ll := range settings.Languages {
					if lang == ll {
						mCountryLanguage.Set(country, lang)
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
	if staticDir != nil && templateDir != nil {
		settings.Config.Embed.Static = true
		settings.Config.Embed.Templates = true
		Static = *staticDir
		Templates = *templateDir
	} else {
		logger.Error("cannot embed static and templates:", staticDir, templateDir)
	}
}

func getTagsAndPrint() {
	h := flag.String("h", "localhost", "Host can be ip or domain name")
	p := flag.String("p", "9313", "Port")
	logs := flag.Bool("logs", false, "overwrite settings.Config.Logs for router /logs")
	monitoring := flag.Bool("monitoring", false, "set settings.Config.Monitoring for prometheus and grafana /metrics")
	docs := flag.Bool("docs", false, "set settings.Config.Docs for prometheus and grafana /docs")
	profiler := flag.Bool("profiler", false, "set settings.Config.Profiler for pprof  /debug/pprof")
	cert := flag.String("cert", "", "certfile")
	key := flag.String("key", "", "keyfile")
	domains := flag.String("domains", "", "domains separated by comma, no www(added automaticaly) to certificates letsencrypt when auto generated")
	flag.Parse()

	if *logs {
		settings.Config.Logs = *logs
	}
	if *monitoring {
		settings.Config.Monitoring = *monitoring
	}
	if *docs {
		settings.Config.Docs = *docs
	}
	if *profiler {
		settings.Config.Profiler = *profiler
	}
	if *cert != "" {
		settings.Config.Cert = *cert
	}
	if *key != "" {
		settings.Config.Key = *key
	}
	if *domains != "" {
		settings.Config.Domains = *domains
	}
	if *p != "9313" {
		settings.Config.Port = *p
	}
	if *h != "localhost" && *h != "127.0.0.1" && *h != "" {
		settings.Config.Host = *h
	} else {
		settings.Config.Host = "localhost"
	}
	host := settings.Config.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := settings.Config.Port
	if port == "" {
		settings.Config.Port = "9313"
		port = "9313"
	}

	logger.Printfs("yl%s", logger.Ascii7)
	logger.Printfs("%s", "-------⚡🚀 http://"+host+":"+port+" 🚀⚡-------")
	if host == "0.0.0.0" || (len(strings.Split(host, ".")) < 4 && host != "localhost") {
		pIp := utils.GetPrivateIp()
		logger.Printfs("HOST IP 0.0.0.0 --> %s", pIp)
	}
}

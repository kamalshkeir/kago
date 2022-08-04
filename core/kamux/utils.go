package kamux

import (
	"embed"
	"encoding/csv"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kamalshkeir/kago/core/settings"
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
	Static = *staticDir
	Templates = *templateDir
}

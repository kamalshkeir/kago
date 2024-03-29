package kamux

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/http/pprof"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kamalshkeir/kago/core/orm"
	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils"
	"github.com/kamalshkeir/kago/core/utils/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var allTemplates = template.New("")

// initTemplatesAndAssets init templates from a folder and download admin skeleton html files
func initTemplatesAndAssets(router *Router) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if !settings.Config.Embed.Static && !settings.Config.Embed.Templates {
			router.cloneTemplatesAndStatic()
		}
	}()
	go func() {
		defer wg.Done()
		router.initDefaultUrls()
	}()
	wg.Wait()
	if settings.Config.Embed.Templates {
		router.AddEmbededTemplates(Templates, settings.TEMPLATE_DIR)
	} else {
		router.AddLocalTemplates(settings.TEMPLATE_DIR)
	}
}

func (router *Router) NewFuncMap(funcName string, function any) {
	if _, ok := functions[funcName]; ok {
		logger.Error("unable to add", funcName, ",already exist")
	} else {
		functions[funcName] = function
	}
}

func (router *Router) ServeLocalDir(dirPath, webPath string) {
	dirPath = filepath.ToSlash(dirPath)
	if webPath[0] != '/' {
		webPath = "/" + webPath
	}
	if webPath[len(webPath)-1] != '/' {
		webPath += "/"
	}
	router.GET(webPath+"*", func(c *Context) {
		http.StripPrefix(webPath, http.FileServer(http.Dir(dirPath))).ServeHTTP(c.ResponseWriter, c.Request)
	})
}

func (router *Router) ServeEmbededDir(pathLocalDir string, embeded embed.FS, webPath string) {
	pathLocalDir = filepath.ToSlash(pathLocalDir)
	if webPath[0] != '/' {
		webPath = "/" + webPath
	}
	if webPath[len(webPath)-1] != '/' {
		webPath += "/"
	}
	toembed_dir, err := fs.Sub(embeded, pathLocalDir)
	if err != nil {
		logger.Error("ServeEmbededDir error=", err)
		return
	}
	toembed_root := http.FileServer(http.FS(toembed_dir))
	router.GET(webPath+"*", func(c *Context) {
		http.StripPrefix(webPath, toembed_root).ServeHTTP(c.ResponseWriter, c.Request)
	})
}

func (router *Router) AddLocalTemplates(pathToDir string) error {
	cleanRoot := filepath.ToSlash(pathToDir)
	pfx := len(cleanRoot) + 1

	err := filepath.Walk(cleanRoot, func(path string, info os.FileInfo, e1 error) error {
		if !info.IsDir() && strings.HasSuffix(path, ".html") {
			if e1 != nil {
				return e1
			}

			b, e2 := os.ReadFile(path)
			if e2 != nil {
				return e2
			}
			name := filepath.ToSlash(path[pfx:])
			t := allTemplates.New(name).Funcs(functions)
			_, e2 = t.Parse(string(b))
			if e2 != nil {
				return e2
			}
		}

		return nil
	})

	return err
}

func (router *Router) AddEmbededTemplates(template_embed embed.FS, rootDir string) error {
	cleanRoot := filepath.ToSlash(rootDir)
	pfx := len(cleanRoot) + 1

	err := fs.WalkDir(template_embed, cleanRoot, func(path string, info fs.DirEntry, e1 error) error {
		if logger.CheckError(e1) {
			return e1
		}
		if !info.IsDir() && strings.HasSuffix(path, ".html") {
			b, e2 := template_embed.ReadFile(path)
			if logger.CheckError(e2) {
				return e2
			}

			name := filepath.ToSlash(path[pfx:])
			t := allTemplates.New(name).Funcs(functions)
			_, e3 := t.Parse(string(b))
			if logger.CheckError(e3) {
				return e2
			}
		}

		return nil
	})

	return err
}

func (router *Router) initDefaultUrls() {
	// prometheus metrics
	if settings.Config.Monitoring {
		router.GET("/metrics", func(c *Context) {
			promhttp.Handler().ServeHTTP(c.ResponseWriter, c.Request)
		})
	}
	// PROFILER
	if settings.Config.Profiler {
		router.GET("/debug/*", func(c *Context) {
			if strings.Contains(c.Request.URL.Path, "profile") {
				pprof.Profile(c.ResponseWriter, c.Request)
				return
			} else if strings.Contains(c.Request.URL.Path, "trace") {
				pprof.Trace(c.ResponseWriter, c.Request)
				return
			}
			pprof.Index(c.ResponseWriter, c.Request)
		})
	}

	// STATIC
	if settings.Config.Embed.Static {
		//EMBED STATIC
		router.ServeEmbededDir(settings.STATIC_DIR, Static, "static")
		if settings.Config.Docs {
			router.ServeEmbededDir(settings.STATIC_DIR+"/docs", Static, "docs")
		}
	} else {
		// LOCAL STATIC
		if _, err := os.Stat(settings.STATIC_DIR); err == nil {
			router.ServeLocalDir(settings.STATIC_DIR, "static")
			if settings.Config.Docs {
				if _, err := os.Stat(settings.STATIC_DIR + "/docs"); err == nil {
					router.ServeLocalDir(settings.STATIC_DIR+"/docs", "docs")
				} else {
					logger.Error(settings.STATIC_DIR+"/docs", "not found")
					os.Exit(0)
				}
			}
		}
	}
	// MEDIA
	media_root := http.FileServer(http.Dir("./" + settings.MEDIA_DIR))
	router.GET(`/`+settings.MEDIA_DIR+`/*`, func(c *Context) {
		http.StripPrefix("/"+settings.MEDIA_DIR+"/", media_root).ServeHTTP(c.ResponseWriter, c.Request)
	})
}

func (router *Router) cloneTemplatesAndStatic() {
	if settings.Config.Embed.Static || settings.Config.Embed.Templates {
		return
	}
	var generated bool

	new_name := "assets"
	if _, err := os.Stat(new_name); err != nil && !settings.Config.Embed.Static && !settings.Config.Embed.Templates {
		// if not generated
		cmd := exec.Command("git", "clone", "https://github.com/"+settings.REPO_USER+"/"+settings.REPO_NAME)
		err := cmd.Run()
		if logger.CheckError(err) {
			return
		}
		err = os.RemoveAll(settings.REPO_NAME + "/.git")
		if err != nil {
			logger.Error("unable to delete", settings.REPO_NAME+"/.git :", err)
		}
		err = os.Rename(settings.REPO_NAME, new_name)
		if err != nil {
			logger.Error("unable to rename", settings.REPO_NAME, err)
		}
		generated = true
	}

	tables := orm.GetAllTables()
	found := false
	for _, t := range tables {
		if t == "users" {
			found = true
		}
	}
	err := orm.Migrate()
	logger.CheckError(err)
	if !found {
		fmt.Printf(logger.Blue, "initial models migrated")
		fmt.Printf(logger.Blue, "you can run 'go run main.go shell' to createsuperuser")
		os.Exit(0)
	}

	if generated && !found {
		fmt.Printf(logger.Green, "assets generated")
		fmt.Printf(logger.Blue, "exec: go run main.go shell -> createsuperuser to create your admin account")
		os.Exit(0)
	} else if generated {
		fmt.Printf(logger.Green, "assets generated")
		os.Exit(0)
	}
}

/* FUNC MAPS */
var functions = template.FuncMap{
	"contains": func(str string, substrings ...string) bool {
		for _, substr := range substrings {
			if strings.Contains(strings.ToLower(str), substr) {
				return true
			}
		}
		return false
	},
	"startWith": func(str string, substrings ...string) bool {
		for _, substr := range substrings {
			if strings.HasPrefix(strings.ToLower(str), substr) {
				return true
			}
		}
		return false
	},
	"finishWith": func(str string, substrings ...string) bool {
		for _, substr := range substrings {
			if strings.HasSuffix(strings.ToLower(str), substr) {
				return true
			}
		}
		return false
	},
	"generateUUID": func() template.HTML {
		uuid, _ := utils.GenerateUUID()
		return template.HTML(uuid)
	},
	"add": func(a int, b int) int {
		return a + b
	},
	"safe": func(str string) template.HTML {
		return template.HTML(str)
	},
	"timeFormat": func(t any) string {
		valueToReturn := ""
		switch v := t.(type) {
		case time.Time:
			if !v.IsZero() {
				valueToReturn = v.Format("2006-01-02T15:04")
			} else {
				valueToReturn = time.Now().Format("2006-01-02T15:04")
			}
		case string:
			if len(v) >= len("2006-01-02T15:04") && strings.Contains(v[:13], "T") {
				p, err := time.Parse("2006-01-02T15:04", v)
				if logger.CheckError(err) {
					valueToReturn = time.Now().Format("2006-01-02T15:04")
				} else {
					valueToReturn = p.Format("2006-01-02T15:04")
				}
			} else {
				if len(v) >= 16 {
					p, err := time.Parse("2006-01-02 15:04", v[:16])
					if logger.CheckError(err) {
						valueToReturn = time.Now().Format("2006-01-02T15:04")
					} else {
						valueToReturn = p.Format("2006-01-02T15:04")
					}
				}
			}
		default:
			if v != nil {
				logger.Error("type of", t, "is not handled,type is:", v)
			}
			valueToReturn = ""
		}
		return valueToReturn
	},
	"date": func(t any) string {
		dString := "02 Jan 2006"
		valueToReturn := ""
		switch v := t.(type) {
		case time.Time:
			if !v.IsZero() {
				valueToReturn = v.Format(dString)
			} else {
				valueToReturn = time.Now().Format(dString)
			}
		case string:
			if len(v) >= len(dString) && strings.Contains(v[:13], "T") {
				p, err := time.Parse(dString, v)
				if logger.CheckError(err) {
					valueToReturn = time.Now().Format(dString)
				} else {
					valueToReturn = p.Format(dString)
				}
			} else {
				if len(v) >= 16 {
					p, err := time.Parse(dString, v[:16])
					if logger.CheckError(err) {
						valueToReturn = time.Now().Format(dString)
					} else {
						valueToReturn = p.Format(dString)
					}
				}
			}
		default:
			if v != nil {
				logger.Error("type of", t, "is not handled,type is:", v)
			}
			valueToReturn = ""
		}
		return valueToReturn
	},
	"slug": func(str string) string {
		if len(str) == 0 {
			return ""
		}
		res, err := utils.ToSlug(str)
		if err != nil {
			return ""
		}
		return res
	},
	"truncate": func(str any, size int) any {
		switch v := str.(type) {
		case string:
			if len(v) > size {
				return v[:size] + "..."
			} else {
				return v
			}
		default:
			return v
		}
	},
	"csrf_token": func(r *http.Request) template.HTML {
		csrf, _ := r.Cookie("csrf_token")
		if csrf != nil {
			return template.HTML(fmt.Sprintf("   <input type=\"hidden\" id=\"csrf_token\" value=\"%s\">   ", csrf.Value))
		} else {
			return template.HTML("")
		}
	},
	"translateFromRequest": func(translation string, request *http.Request) any {
		var lg string
		if language, err := request.Cookie("lang"); err == nil {
			lg = strings.ToLower(language.Value)
		} else {
			lg = "en"
		}

		if data, ok := settings.Translations.Get(lg); ok {
			if v, ok := data[translation]; ok {
				return v
			} else if strings.Contains(translation, ".") {
				sp := strings.Split(translation, ".")
				if len(sp) >= 2 && len(sp) < 4 {
					if d, ok := data[sp[0]]; ok {
						if f, ok := d.(map[string]any)[sp[1]]; ok {
							switch v := f.(type) {
							case string:
								return v
							case map[string]any:
								if vv, ok := v[sp[2]]; ok {
									return vv
								}
							default:
								return "NOT HANDLED"
							}
						}
					}
				}
			}
		} else {
			return "LANGUAGE NOT FOUND FROM COOKIE"
		}

		return "NOT VALID"
	},
	"translateFromLang": func(translation, language string) any {
		if data, ok := settings.Translations.Get(language); ok {
			if v, ok := data[translation]; ok {
				return v
			} else if strings.Contains(translation, ".") {
				sp := strings.Split(translation, ".")
				if len(sp) >= 2 && len(sp) < 4 {
					if d, ok := data[sp[0]]; ok {
						if f, ok := d.(map[string]any)[sp[1]]; ok {
							switch v := f.(type) {
							case string:
								return v
							case map[string]any:
								if vv, ok := v[sp[2]]; ok {
									return vv
								}
							default:
								return "NOT HANDLED"
							}
						}
					}
				}
			}
		} else {
			return "LANGUAGE NOT FOUND FROM COOKIE"
		}

		return "NOT VALID"
	},
}

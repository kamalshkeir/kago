package kamux

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kamalshkeir/kago/core/orm"
	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils"
	"github.com/kamalshkeir/kago/core/utils/logger"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/websocket"
)

var (
	CORSDebug    = false
	ReadTimeout  = 5 * time.Second
	WriteTimeout = 20 * time.Second
	IdleTimeout  = 20 * time.Second
	midwrs       []func(http.Handler) http.Handler
)

// InitServer init the server with midws,
func (router *Router) initServer() {
	port := settings.Config.Port
	var handler http.Handler
	if len(midwrs) != 0 {
		handler = midwrs[0](router)
		for i := 1; i < len(midwrs); i++ {
			handler = midwrs[i](handler)
		}
	} else {
		handler = router
	}
	host := settings.Config.Host

	if host == "" {
		host = "127.0.0.1"
	}

	if port == "" {
		port = "9313"
	}
	// Setup Server
	server := http.Server{
		Addr:         host + ":" + port,
		Handler:      handler,
		ReadTimeout:  ReadTimeout,
		WriteTimeout: WriteTimeout,
		IdleTimeout:  IdleTimeout,
	}
	router.Server = &server
}

func (router *Router) autoServer(tlsconf *tls.Config) {
	port := settings.Config.Port
	var handler http.Handler
	if len(midwrs) != 0 {
		handler = midwrs[0](router)
		for i := 1; i < len(midwrs); i++ {
			handler = midwrs[i](handler)
		}
	} else {
		handler = router
	}
	host := settings.Config.Host

	if host == "" {
		host = "127.0.0.1"
	}

	if port == "" {
		port = "9313"
	}
	// Setup Server
	server := http.Server{
		Addr:         host + ":" + port,
		Handler:      handler,
		ReadTimeout:  ReadTimeout,
		WriteTimeout: WriteTimeout,
		IdleTimeout:  IdleTimeout,
		TLSConfig:    tlsconf,
	}
	router.Server = &server
}

// UseMiddlewares chain global middlewares applied on the router
func (router *Router) UseMiddlewares(midws ...func(http.Handler) http.Handler) {
	midwrs = append(midwrs, midws...)
}

// Run start the server
func (router *Router) Run() {
	if settings.MODE != "barebone" {
		// init templates and assets
		initTemplatesAndAssets(router)
	} else {
		router.initDefaultUrls()
		if settings.Config.Embed.Templates {
			router.AddEmbededTemplates(Templates, settings.TEMPLATE_DIR)
		} else {
			if _, err := os.Stat(settings.TEMPLATE_DIR); err == nil {
				router.AddLocalTemplates(settings.TEMPLATE_DIR)
			}
		}
	}

	tls := router.createAndHandleServerCerts()

	// graceful Shutdown server + db if exist
	go router.gracefulShutdown()

	if tls {
		if err := router.Server.ListenAndServeTLS(settings.Config.Cert, settings.Config.Key); err != http.ErrServerClosed {
			logger.Error("Unable to shutdown the server : ", err)
		} else {
			fmt.Printf(logger.Green, "Server Off !")
		}
	} else {
		if err := router.Server.ListenAndServe(); err != http.ErrServerClosed {
			logger.Error("Unable to shutdown the server : ", err)
		} else {
			fmt.Printf(logger.Green, "Server Off !")
		}
	}
}

func (router *Router) createAndHandleServerCerts() bool {
	host := settings.Config.Host
	domains := settings.Config.Domains
	cert := settings.Config.Cert
	key := settings.Config.Key
	domainsToCertify := map[string]bool{}

	if (cert != "" && key != "") || domains == "" || host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0" {
		router.initServer()
		return false
	} else if domains == "" && cert == "" && key == "" {
		err := checkDomain(host)
		if err != nil || host == "localhost" || host == "127.0.0.1" {
			router.initServer()
			return false
		} else {
			// cree un nouveau single domain for host
			if strings.HasPrefix(host, "www.") {
				domainsToCertify[host[4:]]=true
				domainsToCertify[host]=true
			} else {
				domainsToCertify[host]=true
				domainsToCertify["www."+host]=true
			}
		}
	} else if domains != "" {
		if strings.Contains(domains, ",") {
			// many domaine
			mmap := map[string]uint8{}
			sp := strings.Split(domains, ",")
			for i, d := range sp {
				if d == host {
					continue
				}
				mmap[d] = uint8(i)
			}
			if _, ok := mmap[host]; !ok {
				err := checkDomain(host)
				if err == nil {
					domainsToCertify[host]=true
				}
			}
			for k := range mmap {
				domainsToCertify[k]=true
				if len(strings.Split(k,".")) == 2 && !strings.HasPrefix(k,"www") {
					domainsToCertify["www."+k]=true
				}
			}
		} else {
			sp := strings.Split(domains, ".")
			if strings.HasPrefix(domains, "www.") && domains != host && len(sp) == 3 {
				domainsToCertify[domains[4:]]=true
				domainsToCertify[domains]=true
			} else if domains != host && len(sp) == 2 {
				domainsToCertify[domains]=true
				domainsToCertify["www."+domains]=true
			} else if domains != host && len(sp) == 3 {
				domainsToCertify[domains]=true
			}

			err := checkDomain(host)
			if err == nil {
				sp := strings.Split(host, ".")
				if len(sp) == 2 {
					domainsToCertify[host]=true
					domainsToCertify["www."+host]=true
				} else if len(sp) == 3 && sp[0] == "www" {
					domainsToCertify[host]=true
					domainsToCertify[host[4:]]=true
				} else {
					domainsToCertify[host]=true
				}
			}
		}
	}
	pIP := utils.GetPrivateIp()
	if _,ok := domainsToCertify[pIP];!ok {
		domainsToCertify[pIP]=true
	}
	uniqueDomains := []string{}
	for k := range domainsToCertify {
		uniqueDomains = append(uniqueDomains, k)
	}

	if len(domainsToCertify) > 0 {
		m := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			Cache:      autocert.DirCache("certs"),
			HostPolicy: autocert.HostWhitelist(uniqueDomains...),
		}
		tlsConfig := m.TLSConfig()
		tlsConfig.NextProtos = append([]string{"h2", "http/1.1"}, tlsConfig.NextProtos...) 
		router.autoServer(tlsConfig)
		logger.Printfs("grAuto certified domains: %v", uniqueDomains)
	}
	return true
}

func checkDomain(name string) error {
	switch {
	case len(name) == 0:
		return nil
	case len(name) > 255:
		return fmt.Errorf("cookie domain: name length is %d, can't exceed 255", len(name))
	}
	var l int
	for i := 0; i < len(name); i++ {
		b := name[i]
		if b == '.' {
			switch {
			case i == l:
				return fmt.Errorf("cookie domain: invalid character '%c' at offset %d: label can't begin with a period", b, i)
			case i-l > 63:
				return fmt.Errorf("cookie domain: byte length of label '%s' is %d, can't exceed 63", name[l:i], i-l)
			case name[l] == '-':
				return fmt.Errorf("cookie domain: label '%s' at offset %d begins with a hyphen", name[l:i], l)
			case name[i-1] == '-':
				return fmt.Errorf("cookie domain: label '%s' at offset %d ends with a hyphen", name[l:i], l)
			}
			l = i + 1
			continue
		}
		if !(b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '-' || b >= 'A' && b <= 'Z') {
			// show the printable unicode character starting at byte offset i
			c, _ := utf8.DecodeRuneInString(name[i:])
			if c == utf8.RuneError {
				return fmt.Errorf("cookie domain: invalid rune at offset %d", i)
			}
			return fmt.Errorf("cookie domain: invalid character '%c' at offset %d", c, i)
		}
	}
	switch {
	case l == len(name):
		return fmt.Errorf("cookie domain: missing top level domain, domain can't end with a period")
	case len(name)-l > 63:
		return fmt.Errorf("cookie domain: byte length of top level domain '%s' is %d, can't exceed 63", name[l:], len(name)-l)
	case name[l] == '-':
		return fmt.Errorf("cookie domain: top level domain '%s' at offset %d begins with a hyphen", name[l:], l)
	case name[len(name)-1] == '-':
		return fmt.Errorf("cookie domain: top level domain '%s' at offset %d ends with a hyphen", name[l:], l)
	case name[l] >= '0' && name[l] <= '9':
		return fmt.Errorf("cookie domain: top level domain '%s' at offset %d begins with a digit", name[l:], l)
	}
	return nil
}

// ServeHTTP serveHTTP by handling methods,pattern,and params
func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const key utils.ContextKey = "params"
	c := &Context{Request: r, ResponseWriter: w, Params: map[string]string{}}
	var allRoutes []Route
	switch r.Method {
	case "GET":
		if strings.Contains(r.URL.Path, "/ws/") {
			allRoutes = router.Routes[WS]
		} else if strings.Contains(r.URL.Path, "/sse/") {
			allRoutes = router.Routes[SSE]
		} else {
			allRoutes = router.Routes[GET]
		}
	case "POST":
		allRoutes = router.Routes[POST]
	case "PUT":
		allRoutes = router.Routes[PUT]
	case "PATCH":
		allRoutes = router.Routes[PATCH]
	case "DELETE":
		allRoutes = router.Routes[DELETE]
	case "HEAD":
		allRoutes = router.Routes[HEAD]
	case "OPTIONS":
		allRoutes = router.Routes[OPTIONS]
	default:
		c.Status(http.StatusBadRequest).Text("Method Not Allowed")
		return
	}

	if len(allRoutes) > 0 {
		for _, rt := range allRoutes {
			// match route
			if matches := rt.Pattern.FindStringSubmatch(c.URL.Path); len(matches) > 0 {
				// add params
				paramsValues := matches[1:]
				if names := rt.Pattern.SubexpNames(); len(names) > 0 {
					for i, name := range rt.Pattern.SubexpNames()[1:] {
						if name != "" {
							c.Params[name] = paramsValues[i]
						}
					}
					ctx := context.WithValue(c.Request.Context(), key, c.Params)
					c.Request = r.WithContext(ctx)
				}
				if rt.WsHandler != nil {
					// WS
					rt.Method = r.Method
					handleWebsockets(c, rt)
					return
				} else {
					// HTTP
					rt.Method = r.Method
					handleHttp(c, rt)
					return
				}
			}
		}
	}
	router.DefaultRoute(c)
}

// Graceful Shutdown
func (router *Router) gracefulShutdown() {
	err := utils.GracefulShutdown(func() error {
		// Close databases
		if err := orm.ShutdownDatabases(); err != nil {
			logger.Error("unable to shutdown databases:", err)
		} else {
			fmt.Printf(logger.Blue, "Databases Closed")
		}
		// Shutdown server
		router.Server.SetKeepAlivesEnabled(false)
		err := router.Server.Shutdown(context.Background())
		if logger.CheckError(err) {
			return err
		}
		return nil
	})
	if logger.CheckError(err) {
		os.Exit(1)
	}
}

func ParamsHandleFunc(r *http.Request) (map[string]string, bool) {
	const key utils.ContextKey = "params"
	params, ok := r.Context().Value(key).(map[string]string)
	if ok {
		return params, true
	}
	return nil, false
}

func adaptParams(url string) string {
	if strings.Contains(url, ":") {
		urlElements := strings.Split(url, "/")
		urlElements = urlElements[1:]
		for i, elem := range urlElements {
			// named types
			if elem[0] == ':' {
				urlElements[i] = `(?P<` + elem[1:] + `>\w+)`
			} else if strings.Contains(elem, ":") {
				nameType := strings.Split(elem, ":")
				name := nameType[0]
				name_type := nameType[1]
				switch name_type {
				case "str":
					//urlElements[i] = `(?P<` + name + `>\w+)+\/?`
					urlElements[i] = `(?P<` + name + `>\w+)`
				case "int":
					urlElements[i] = `(?P<` + name + `>\d+)`
				case "slug":
					urlElements[i] = `(?P<` + name + `>[a-z0-9]+(?:-[a-z0-9]+)*)`
				case "float":
					urlElements[i] = `(?P<` + name + `>[-+]?([0-9]*\.[0-9]+|[0-9]+))`
				default:
					urlElements[i] = `(?P<` + name + `>[a-z0-9]+(?:-[a-z0-9]+)*)`
				}
			}
		}
		join := strings.Join(urlElements, "/")
		if !strings.HasSuffix(join, "*") {
			join += "(|/)?$"
		}
		return "^/" + join
	}

	if url[len(url)-1] == '*' {
		return url
	} else {
		return "^" + url + "(|/)?$"
	}
}

func checkSameSite(c Context) bool {
	privateIp := ""
	origin := c.Request.Header.Get("Origin")
	if CORSDebug {
		logger.Info("ORIGIN", origin)
		logger.Info("HOST:", settings.Config.Host)
		logger.Info("PORT:", settings.Config.Port)
		logger.Info("DOMAINS:", settings.Config.Domains)
	}
	if origin == "" {
		return false
	}

	if len(Origines) > 0 {
		for _, o := range Origines {
			if strings.Contains(origin, o) || o == "*" {
				return true
			}
		}
	}

	host := settings.Config.Host
	if host == "" || host == "localhost" || host == "127.0.0.1" {
		if strings.Contains(origin, "localhost") {
			host = "localhost"
		} else if strings.Contains(origin, "127.0.0.1") {
			host = "127.0.0.1"
		} else {
			host = "127.0.0.1"
		}
	}
	port := settings.Config.Port
	if port != "" {
		port = ":" + port
	}
	privateIp = utils.GetPrivateIp()
	if utils.StringContains(c.Request.RemoteAddr, host, "localhost", "127.0.0.1", privateIp) {
		return true
	}

	if CORSDebug {
		logger.Info("ORIGIN of remote ", c.Request.RemoteAddr, "is:", origin)
		logger.Info("HOST:", host)
		logger.Info("PORT:", port)
		logger.Info("DOMAINS:", settings.Config.Domains)
	}

	if settings.Config.Domains != "" {
		if strings.Contains(settings.Config.Domains, ",") {
			sp := strings.Split(settings.Config.Domains, ",")
			for _, s := range sp {
				if strings.Contains(origin, s) {
					return true
				}
			}
		} else {
			if strings.Contains(origin, settings.Config.Domains) {
				return true
			}
		}
	}

	foundInPrivateIps := false
	if host != "localhost" && host != "127.0.0.1" {
		if strings.Contains(origin, host) {
			foundInPrivateIps = true
		} else if strings.Contains(origin, privateIp) {
			foundInPrivateIps = true
		} else {
			logger.Info("origin:", origin, "not equal to privateIp:", privateIp+port)
		}
	}

	sp := strings.Split("host", ".")
	if utils.StringContains(origin, host, "localhost"+port, "127.0.0.1"+port) || foundInPrivateIps || (len(sp) < 4 && host != "localhost" && host != "127.0.0.1") {
		return true
	} else {
		return false
	}
}

func handleWebsockets(c *Context, rt Route) {
	if checkSameSite(*c) {
		// same site
		websocket.Handler(func(conn *websocket.Conn) {
			conn.MaxPayloadBytes = 10 << 20
			if conn.IsServerConn() {
				ctx := &WsContext{
					Ws:     conn,
					Params: make(map[string]string),
					Route:  rt,
				}
				rt.WsHandler(ctx)
				return
			}
		}).ServeHTTP(c.ResponseWriter, c.Request)
		return
	} else {
		// cross
		if len(rt.AllowedOrigines) == 0 {
			c.Status(http.StatusBadRequest).Text("you are not allowed cross origin for this url")
			return
		} else {
			allowed := false
			for _, dom := range rt.AllowedOrigines {
				if strings.Contains(c.Request.Header.Get("Origin"), dom) {
					allowed = true
				}
			}
			if allowed {
				websocket.Handler(func(conn *websocket.Conn) {
					conn.MaxPayloadBytes = 10 << 20
					if conn.IsServerConn() {
						ctx := &WsContext{
							Ws:     conn,
							Params: make(map[string]string),
							Route:  rt,
						}
						rt.WsHandler(ctx)
						return
					}
				}).ServeHTTP(c.ResponseWriter, c.Request)
				return
			} else {
				c.Status(http.StatusBadRequest).Text("you are not allowed to access this route from cross origin")
				return
			}
		}
	}
}

func handleHttp(c *Context, rt Route) {
	switch rt.Method {
	case "GET":
		if rt.Method == "SSE" {
			sseHeaders(c)
		}
		rt.Handler(c)
		return
	case "SSE":
		sseHeaders(c)
		rt.Handler(c)
		return
	case "HEAD", "OPTIONS":
		rt.Handler(c)
		return
	default:
		// check cross origin
		if checkSameSite(*c) {
			// same site
			rt.Handler(c)
			return
		} else {
			// cross origin
			if len(rt.AllowedOrigines) == 0 {
				c.Status(http.StatusBadRequest).Text("cross origin not allowed")
				return
			} else {
				if rt.AllowedOrigines[0] == "*" {
					rt.Handler(c)
					return
				}

				allowed := false
				for _, dom := range rt.AllowedOrigines {
					if strings.Contains(c.Request.Header.Get("Origin"), dom) {
						allowed = true
					}
				}
				if allowed {
					rt.Handler(c)
					return
				} else {
					c.Status(http.StatusBadRequest).Text("you are not allowed cross origin this url")
					return
				}
			}
		}
	}

}

func sseHeaders(c *Context) {
	o := strings.Join(Origines, ",")
	c.SetHeader("Access-Control-Allow-Origin", o)
	c.SetHeader("Access-Control-Allow-Headers", "Content-Type")
	c.SetHeader("Cache-Control", "no-cache")
	c.SetHeader("Connection", "keep-alive")
}

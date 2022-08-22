package kamux

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/kamalshkeir/kago/core/orm"
	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils"
	"github.com/kamalshkeir/kago/core/utils/logger"
	"github.com/kamalshkeir/kago/core/utils/reverseproxy"
	"golang.org/x/net/websocket"
)

var (
	ReadTimeout=  5 * time.Second
	WriteTimeout= 20 * time.Second
	IdleTimeout= 20 * time.Second
)


var midwrs []func(http.Handler) http.Handler

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
			if _,err := os.Stat(settings.TEMPLATE_DIR);err == nil {
				router.AddLocalTemplates(settings.TEMPLATE_DIR)
			}
		}
	}
	
	// init server
	router.initServer()
	// graceful Shutdown server + db if exist
	go router.gracefulShutdown()

	if err := router.Server.ListenAndServe(); err != http.ErrServerClosed {
		logger.Error("Unable to shutdown the server : ", err)
	} else {
		fmt.Printf(logger.Green, "Server Off !")
	}
}


var proxies = map[string]string{}

func (router *Router) Proxy(from string, to string) {
	proxies[from]=to
}


// RunTLS start the server TLS
func (router *Router) RunTLS(certFile string, keyFile string) {
	if settings.MODE != "barebone" {
		// init templates and assets
		initTemplatesAndAssets(router)
	} else {
		router.initDefaultUrls()
		if settings.Config.Embed.Templates {
			router.AddEmbededTemplates(Templates, settings.TEMPLATE_DIR)
		} else {
			if _,err := os.Stat(settings.TEMPLATE_DIR);err == nil { 
				router.AddLocalTemplates(settings.TEMPLATE_DIR)
			}
		}
	}
	if certFile != "" && keyFile != "" {
		settings.Config.Cert=certFile
		settings.Config.Key=keyFile
	}

	
	// init server
	router.initServer()
	// graceful Shutdown server + db if exist
	go router.gracefulShutdown()
	if settings.Config.Cert != "" && settings.Config.Key != "" && settings.Proxy {
		// proxy
		go func() {
			http.ListenAndServeTLS(settings.Config.Host+":"+settings.Config.Port,settings.Config.Cert,settings.Config.Key,http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if utils.StringContains(r.Host,"localhost:9313","127.0.0.1:9313","0.0.0.0:9313") {
					path,err := url.Parse("http://localhost:9313")
					if err != nil {
						fmt.Println(err)
						return
					}
					proxy := reverseproxy.NewReverseProxy(path)
					proxy.ServeHTTP(w,r)
				} else {
					for dom,urll := range proxies {
						if strings.Contains(r.Host,dom) {
							path,err := url.Parse(urll)
							if err != nil {
								fmt.Println(err)
								return
							}
							proxy := reverseproxy.NewReverseProxy(path)
							proxy.ServeHTTP(w,r)
						}
					}
				}
			}))
		}()
	}

	if settings.Proxy {
		router.Server.Addr="localhost:9313"
		if err := router.Server.ListenAndServe(); err != http.ErrServerClosed {
			logger.Error("Unable to shutdown the server : ", err)
		} else {
			fmt.Printf(logger.Green, "Server Off !")
		}
	} else {
		if err := router.Server.ListenAndServeTLS(settings.Config.Cert,settings.Config.Key); err != http.ErrServerClosed {
			logger.Error("Unable to shutdown the server : ", err)
		} else {
			fmt.Printf(logger.Green, "Server Off !")
		}
	}
	
}

// ServeHTTP serveHTTP by handling methods,pattern,and params
func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
						c.Params[name] = paramsValues[i]
					}
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
		return "^/" + strings.Join(urlElements, "/") + "(|/)?$"
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
	if origin == "" {
		return false
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

	foundInPrivateIps := false
	if host != "localhost" && host != "127.0.0.1"{
		privateIp = utils.GetPrivateIp()
		if strings.Contains(origin,host) {
			foundInPrivateIps = true
		} else if strings.Contains(origin, privateIp) {
			foundInPrivateIps = true
		} else {
			logger.Info("origin:",origin,"not equal to privateIp:",privateIp+port)
		}
	}

	sp := strings.Split("host",".")
	if utils.StringContains(origin, host, "localhost"+port, "127.0.0.1"+port) || foundInPrivateIps || (len(sp)<4 && host != "localhost" && host != "127.0.0.1") {
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
	case "HEAD","OPTIONS":
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
				c.Status(http.StatusBadRequest).Text("no cross origin not allowed")
				return
			} else {
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
	c.SetHeader("Access-Control-Allow-Origin", "*")
	c.SetHeader("Access-Control-Allow-Headers", "Content-Type")
	c.SetHeader("Content-Type", "text/event-stream")
	c.SetHeader("Cache-Control", "no-cache")
	c.SetHeader("Connection", "keep-alive")
}

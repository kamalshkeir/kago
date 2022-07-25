package kamux

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kamalshkeir/kago/core/orm"
	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/shell"
	"github.com/kamalshkeir/kago/core/utils"
	"github.com/kamalshkeir/kago/core/utils/logger"
	"golang.org/x/net/websocket"
)


var midwrs []func(http.Handler) http.Handler

// InitServer init the server with midws,
func (router *Router) initServer() {
	var handler http.Handler
	if len(midwrs) != 0 {
		handler = midwrs[0](router)
		for i := 1; i < len(midwrs); i++ {
			handler = midwrs[i](handler)
		}
	} else {
		handler = router
	}
	host := settings.GlobalConfig.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := settings.GlobalConfig.Port
	if port == "" {
		port = "9313"
	}
	// Setup Server
	server := http.Server{
		Addr:         host + ":" + port,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 20 * time.Second,
		IdleTimeout:  20 * time.Second,
	}	
	router.Server=&server
}

// UseMiddlewares chain global middlewares applied on the router
func (router *Router) UseMiddlewares(midws ...func(http.Handler) http.Handler) {
	midwrs = append(midwrs, midws...)
}


// Run start the server
func (router *Router) Run() {
	// init orm shell
	if shell.InitShell() {os.Exit(0)}
	// init templates and assets
	initTemplatesAndAssets(router)
	// init server
	router.initServer()
	// graceful Shutdown server + db if exist
	go router.gracefulShutdown()

	if err := router.Server.ListenAndServe(); err != http.ErrServerClosed {
		logger.Error("Unable to shutdown the server : ",err)
	} else {
		fmt.Printf(logger.Green,"Server Off !")
	}
}


// RunTLS start the server TLS
func (router *Router) RunTLS(certFile string,keyFile string) {
	// init orm shell
	if shell.InitShell() {os.Exit(0)}
	// init templates and assets
	initTemplatesAndAssets(router)
	// init server
	router.initServer()
	// graceful Shutdown server + db if exist
	go router.gracefulShutdown()

	if err := router.Server.ListenAndServeTLS(certFile,keyFile); err != http.ErrServerClosed {
		logger.Error("Unable to shutdown the server : ",err)
	} else {
		fmt.Printf(logger.Green,"Server Off !")
	}
}

// Graceful Shutdown
func (router *Router) gracefulShutdown() {
	err := utils.GracefulShutdown(func() error {
		// Close databases
		if err := orm.ShutdownDatabases();err != nil {
			logger.Error("unable to shutdown databases:",err)
		} else {
			fmt.Printf(logger.Blue,"Databases Closed")
		}
		// Shutdown server
		router.Server.SetKeepAlivesEnabled(false)
		err := router.Server.Shutdown(context.Background())
		if logger.CheckError(err) {return err}
		return nil
	})
	if logger.CheckError(err) {os.Exit(1)}
}


// ServeHTTP serveHTTP by handling methods,pattern,and params
func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := &Context{Request: r, ResponseWriter: w, Params: map[string]string{}}
	var allRoutes []Route

	switch r.Method {
	case "GET":
		if strings.Contains(r.URL.Path,"/ws/") {
			allRoutes = router.Routes[WS]
		} else if strings.Contains(r.URL.Path,"/sse/") {
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
	default:
		c.Status(http.StatusBadRequest).Text("Method Not Allowed .")
		return
	}

	if len(allRoutes) > 0 {
		for _, rt := range allRoutes {
			// match route
			if matches := rt.Pattern.FindStringSubmatch(c.URL.Path); len(matches) > 0 {
				// add params
				paramsValues := matches[1:]
				if names := rt.Pattern.SubexpNames();len(names) > 0 {
					for i,name := range rt.Pattern.SubexpNames()[1:] {
						c.Params[name]=paramsValues[i]
					}
				}
				if rt.WsHandler != nil {
					// WS 
					handleWebsockets(c,rt)
					return
				} else {
					// HTTP
					handleHttp(c,rt)
					return
				}
			}
		}
	}
	router.DefaultRoute(c)
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
		return "^/"+strings.Join(urlElements, "/")+"(|/)?$"
	} 

	if url[len(url)-1] == '*' {
		return url
	} else {
		return "^"+url+"(|/)?$"
	}
}

func checkSameSite(c Context) bool {
	origin := c.Request.Header.Get("Origin")
	if origin == "" {
		return false
	}
	host := settings.GlobalConfig.Host
	if host == "" || host == "localhost" || host == "127.0.0.1" {
		if strings.Contains(origin,"localhost") {
			host="localhost"
		} else if strings.Contains(origin,"127.0.0.1") {
			host="127.0.0.1"
		}
	}
	port := settings.GlobalConfig.Port
	if port != "" {
		port=":"+port
	}
	if strings.Contains(origin,host+port) {
		return true
	} else {
		return false
	}
}

func handleWebsockets(c *Context ,rt Route) {
	if checkSameSite(*c) {
		// same site
		websocket.Handler(func(conn *websocket.Conn) {
			conn.MaxPayloadBytes = 10 << 20
			if conn.IsServerConn() {
				ctx := &WsContext{
					Ws: conn,
					Params: make(map[string]string),
					Route: rt,
				}
				rt.WsHandler(ctx)
				return
			}
		}).ServeHTTP(c.ResponseWriter,c.Request)
		return
	} else {
		// cross
		if len(rt.AllowedOrigines) == 0 {
			c.Status(http.StatusBadRequest).Text("you are not allowed cross origin for this url")
			return
		} else {
			allowed := false
			for _,dom := range rt.AllowedOrigines {
				if strings.Contains(c.Request.Header.Get("Origin"),dom) {
					allowed=true
				}
			}
			if allowed {
				websocket.Handler(func(conn *websocket.Conn) {
					conn.MaxPayloadBytes = 10 << 20
					if conn.IsServerConn() {
						ctx := &WsContext{
							Ws: conn,
							Params: make(map[string]string),
							Route: rt,
						}
						rt.WsHandler(ctx)
						return
					}
				}).ServeHTTP(c.ResponseWriter,c.Request)
				return
			} else {
				c.Status(http.StatusBadRequest).Text("you are not allowed to access this route from cross origin")
				return
			}
		}
	}
}

func handleHttp(c *Context,rt Route) {
	if rt.Method == "GET" {
		if rt.Method == "SSE" {
			sseHeaders(c)
		}
		rt.Handler(c)
		return
	}
	// check cross origin
	if checkSameSite(*c) {
		// same site
		rt.Handler(c)
		return
	} else if rt.Method == "SSE" {
		sseHeaders(c)
		rt.Handler(c)
		return
	} else {
		// cross origin
		if len(rt.AllowedOrigines) == 0 {
			c.Status(http.StatusBadRequest).Text("you are not allowed cross origin for this url")
			return
		} else {
			allowed := false
			for _,dom := range rt.AllowedOrigines {
				if strings.Contains(c.Request.Header.Get("Origin"),dom) {
					allowed=true
				}
			}
			if allowed {
				rt.Handler(c)
				return
			} else {
				c.Status(http.StatusBadRequest).Text("you are not allowed to access this route from cross origin")
				return
			}
		}
	}
}

func sseHeaders(c *Context) {
	c.ResponseWriter.Header().Set("Access-Control-Allow-Origin", "*")
    c.ResponseWriter.Header().Set("Access-Control-Allow-Headers", "Content-Type")
    c.ResponseWriter.Header().Set("Content-Type", "text/event-stream")
    c.ResponseWriter.Header().Set("Cache-Control", "no-cache")
    c.ResponseWriter.Header().Set("Connection", "keep-alive")
}




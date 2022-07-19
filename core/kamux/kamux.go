package kamux

import (
	"flag"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/kamalshkeir/kago/core/orm"
	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils/logger"
	"github.com/kamalshkeir/kago/core/utils/shell"
	"golang.org/x/net/websocket"
)

const (
	GET int = iota
	POST
	PUT
	PATCH
	DELETE
	WS
	SSE
)
var methods = map[int]string{
	GET:"GET",
	POST:"POST",
	PUT:"PUT",
	PATCH:"PATCH",
	DELETE:"DELETE",
	WS:"WS",
	SSE:"SSE",
}
// Handler
type Handler func(c *Context)
type WsHandler func(c *WsContext)

// Router
type Router struct {
	Routes       map[int][]Route
	DefaultRoute Handler
	Server *http.Server
}

// Route
type Route struct {
	Method  string
	Pattern *regexp.Regexp
	Handler 
	WsHandler
	Clients map[string]*websocket.Conn
	AllowedOrigines []string
}



// New Create New Router from env file default: '.env'
func New(envFiles ...string) *Router {
	var wg sync.WaitGroup
	app := &Router{
		Routes: map[int][]Route{},
		DefaultRoute: func(c *Context) {
			c.Text(404,"Page Not Found")
		},
	}
	wg.Add(1)
	// Load Env
	go func(envFiles ...string) {
		if len(envFiles) > 0 {
			app.LoadEnv(envFiles...)
		} else {
			if _, err := os.Stat(".env"); os.IsNotExist(err) {
				if os.Getenv("DB_NAME") == "" && os.Getenv("DB_DSN") == "" {
					logger.Warn("Environment variables not loaded, you can copy it from generated assets folder and rename it to .env, or set them manualy")
				}
			} else {
				app.LoadEnv(".env")
			}
		}
		wg.Done()
	}(envFiles...)
	
	// check flags
	wg.Add(1)
	go func() {
		h := flag.String("h","localhost","overwrite host")
		p := flag.String("p","9313","overwrite port number")
		logs := flag.Bool("logs",false,"overwrite settings.GlobalConfig.Logs for router /logs")
		monitoring := flag.Bool("monitoring",false,"set settings.GlobalConfig.Monitoring for prometheus and grafana /metrics")
		docs := flag.Bool("docs",false,"set settings.GlobalConfig.Docs for prometheus and grafana /docs")
		profiler := flag.Bool("profiler",false,"set settings.GlobalConfig.Profiler for pprof  /debug/pprof")
		flag.Parse()
		
		settings.GlobalConfig.Logs=*logs
		settings.GlobalConfig.Monitoring=*monitoring
		settings.GlobalConfig.Docs=*docs
		settings.GlobalConfig.Profiler=*profiler

		if *p != "9313" {
			settings.GlobalConfig.Port=*p
		}
		if *h != "localhost" && *h != "127.0.0.1" && *h != "" {
			settings.GlobalConfig.Host=*h
		} else {
			settings.GlobalConfig.Host="localhost"
		}
		wg.Done()
	}()
	

	// init orm
	wg.Add(1)
	go func() {
		err := orm.InitDB()
		if err != nil {
			if os.Getenv("DB_NAME") == "" && os.Getenv("DB_DSN") == "" {
				logger.Warn("Environment variables not loaded, you can copy it from generated assets folder and rename it to .env, or set them manualy")
			} else {
				logger.Error(err)
			}
		} else {
			// init orm shell
			if shell.InitShell() {os.Exit(0)}
		}
		if len(orm.GetAllTables()) > 0 {
			// migrate initial models
			err := orm.Migrate()
			logger.CheckError(err)
		}
		wg.Done()
	}()
	wg.Wait()
	return app
}

// handle a route
func (router *Router) handle(method int,pattern string, handler Handler,wshandler WsHandler,allowed []string) {
	re := regexp.MustCompile(adaptParams(pattern))
	route := Route{Method: methods[method],Pattern: re, Handler: handler, WsHandler: wshandler,Clients: nil,AllowedOrigines: []string{}}
	if len(allowed) > 0 && method != GET {
		route.AllowedOrigines = append(route.AllowedOrigines, allowed...)
	}
	if method == WS {
		route.Clients=map[string]*websocket.Conn{}
	}
	if _,ok := router.Routes[method];!ok {
		router.Routes[method]=[]Route{}
	}
	if len(router.Routes[method]) == 0 {
		router.Routes[method] = append(router.Routes[method], route)
		return
	}
	for i,rt := range router.Routes[method] {
		if rt.Pattern.String() == re.String() {
			router.Routes[method]=append(router.Routes[method][:i],router.Routes[method][i+1:]...)
		} 
	}
	router.Routes[method] = append(router.Routes[method], route)
}

// Get handle GET to a route
func (router *Router) Get(pattern string, handler Handler) {
	router.handle(GET,pattern,handler,nil,nil)
}

// Post handle POST to a route
func (router *Router) Post(pattern string, handler Handler, allowed_origines ...string) {
	router.handle(POST,pattern,handler,nil,allowed_origines)
}

// Put handle PUT to a route
func (router *Router) Put(pattern string, handler Handler, allowed_origines ...string) {
	router.handle(PUT,pattern,handler,nil,allowed_origines)
}

// Patch handle PATCH to a route
func (router *Router) Patch(pattern string, handler Handler, allowed_origines ...string) {
	router.handle(PATCH,pattern,handler,nil,allowed_origines)
}

// Delete handle DELETE to a route
func (router *Router) Delete(pattern string, handler Handler, allowed_origines ...string) {
	router.handle(DELETE,pattern,handler,nil,allowed_origines)
}

// WS handle WS connection on a pattern
func (router *Router) WS(pattern string, wsHandler WsHandler, allowed_origines ...string) {
	router.handle(WS,pattern,nil,wsHandler,allowed_origines)
}

// Delete handle DELETE to a route
func (router *Router) SSE(pattern string, handler Handler, allowed_origines ...string) {
	router.handle(SSE,pattern,handler,nil,allowed_origines)
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
		c.Text(http.StatusBadRequest,"Method Not Allowed .")
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
		splited := strings.Split(url, "/")
		splited = splited[1:]
		for i, s := range splited {
			if strings.Contains(s, ":") {
				nameType := strings.Split(s, ":")
				name := nameType[0]
				name_type := nameType[1]
				switch name_type {
				case "string":
					//splited[i] = `(?P<` + name + `>\w+)+\/?`
					splited[i] = `(?P<` + name + `>\w+)`
				case "int":
					splited[i] = `(?P<` + name + `>\d+)`
				case "slug":
					splited[i] = `(?P<` + name + `>[a-z0-9]+(?:-[a-z0-9]+)*)` 
				case "float":
					splited[i] = `(?P<` + name + `>[-+]?([0-9]*\.[0-9]+|[0-9]+))` 
				default:
					splited[i] = `(?P<` + name + `>[a-z0-9]+(?:-[a-z0-9]+)*)` 
				}
			}
		}
		return "^/"+strings.Join(splited, "/")+"(|/)?$"
	} 

	if strings.Contains(url,"^") {
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
			c.Text(http.StatusBadRequest,"you are not allowed cross origin for this url")
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
				c.Text(http.StatusBadRequest,"you are not allowed to access this route from cross origin")
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
			c.Text(http.StatusBadRequest,"you are not allowed cross origin for this url")
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
				c.Text(http.StatusBadRequest,"you are not allowed to access this route from cross origin")
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

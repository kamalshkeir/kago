<div align="center">
	<img src="https://user-images.githubusercontent.com/54605903/183331771-50e1bb95-2a75-4472-aebc-14e6b44f2a98.png" width="auto" style="margin:0 auto 0 auto;"/>
</div>
<br>

<div align="center">
	<img src="https://img.shields.io/github/go-mod/go-version/kamalshkeir/kago" width="auto" height="20px">
	<img src="https://img.shields.io/github/languages/code-size/kamalshkeir/kago" width="auto" height="20px">
	<img src="https://img.shields.io/badge/License-BSD%20v3-blue.svg" width="auto" height="20px">
	<img src="https://img.shields.io/badge/License-BSD%20v3-blue.svg" width="auto" height="20px">
	<img src="https://img.shields.io/github/v/tag/kamalshkeir/kago" width="auto" height="20px">
	<img src="https://img.shields.io/github/stars/kamalshkeir/kago?style=social" width="auto" height="20px">
	<img src="https://img.shields.io/github/forks/kamalshkeir/kago?style=social" width="auto" height="20px">
</div>
<br>
<div align="center">
	<a href="https://kamalshkeir.dev" target="_blank">
		<img src="https://img.shields.io/badge/my_portfolio-000?style=for-the-badge&logo=ko-fi&logoColor=white" width="auto" height="32px">
	</a>
	<a href="https://www.linkedin.com/in/kamal-shkeir/">
		<img src="https://img.shields.io/badge/linkedin-0A66C2?style=for-the-badge&logo=linkedin&logoColor=white" width="auto" height="30px">
	</a>
	<a href="https://www.buymeacoffee.com/kamalshkeir" target="_blank"><img src="https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png" alt="Buy Me A Coffee" width="auto" height="32px" ></a>

	
</div>

###### Join our [discussions here](https://github.com/kamalshkeir/kago/discussions/1)
---


# KaGo Web Framework 
<br>
KaGo is a Django like web framework, that encourages clean and rapid development.

You can start a new project using 2 lines of code, run a high performance server with CRUD Admin Dashboard, interactive shell, easy and performant ORM, AutoSSL and lots more...

List of packages extracted from kago in order to make it more composable :
- ORM [Korm](https://github.com/kamalshkeir/korm)
- Router [Kmux](https://github.com/kamalshkeir/kmux)
- Network [KSbus](https://github.com/kamalshkeir/ksbus)
- Safemap [Kmap](https://github.com/kamalshkeir/kmap)
- EnvLoader [Kenv](https://github.com/kamalshkeir/kenv)

Quick List of latest features :
- <strong>NEW :</strong>  orm.AutoMigrate generate query to file instead of executing it directly, you can execute it later using go run main.go shell -> migrate
- <strong>NEW :</strong>  orm.AddTrigger and orm.DropTrigger, [examples](#orm-triggers)
- <strong>NEW :</strong>  [Admin Search and OrderBy](#admin-search)
- <strong>NEW :</strong>  [Auto SSL Letsencrypt Certificates](#run-https-letsencrypt-in-production-using-env-vars-and-tags) and keep them up to date (auto renew 1 month before expire)
- [BareBone Mode](#barebone-router-no-assets-cloned) Router Only
- ORM handle coakroachdb in addition to sqlite, postgres, mysql and mariadb (check Performance at the bottom of this readme)
- [Watcher/Auto-Reloader](#watcher-or-auto-reloader)
- [orm.AutoMigrate](#automigrate-usage) will handle all your migrations from a struct model, if you remove a field from the migrated struct, you will be prompted to do the migration, it can handle foreign_keys, checks, indexes,...
- Fully editable [CRUD Admin Dashboard](#generated-admin-dashboard) (assets folder)
- Realtime [Logs](#logs) at `/logs` running with flag `go run main.go --logs`
- Convenient [Router](#routing) that handle params with regex type checking, Websockets and SSE protocols also 
- [Interactive Shell](#shell) `go run main.go shell`
- Maybe the easiest [ORM and SQL builder](#orm) using go generics (working with sqlite, mysql,postgres and coakroach)
- OpenApi [docs](#openapi-documentation-ready-if-enabled-using-flags-or-settings-vars) at `/docs` running with flag `go run main.go --docs` with dedicated package docs to help you manipulate docs.json file
- AES encryption for authentication sessions using package encryptor
- Argon2 hashing using package hash
- [EnvLoader](#env-loader) the fastest package envloader to load directly env to struct
- [Internal Eventbus](#eventbus-internal) using go routines and channels (very powerful), use cases are endless
- [Monitoring Prometheus/grafana](#grafana-with-prometheus-monitoring) `/metrics` running with flag `go run main.go --monitoring`
- [Profiler golang official](#pprof-official-golang-profiling-tools) debug pprof `/debug/pprof/(heap|profile\trace)` running with flag `go run main.go --profiler`
- [Embed](#build-single-binary-with-all-static-and-html-files) your application static and template files
- Ready to use Progressive Web App Support (pure js, without workbox)


Join our  [discussions here](https://github.com/kamalshkeir/kago/discussions/1)

---
# Installation
##### get the last version

```sh
go get -u github.com/kamalshkeir/kago@v1.2.5
```

# Drivers moved outside this package to not get them all in your go.mod file
go get github.com/kamalshkeir/sqlitedriver
go get github.com/kamalshkeir/pgdriver
go get github.com/kamalshkeir/mysqldriver

---

# Quick start
##### Router + Admin + Database + Shell

```go
package main

import (
	"github.com/kamalshkeir/kago"
	"github.com/kamalshkeir/sqlitedriver"
)

func main() {
	sqlitedriver.Use() // use one of this drivers
	mysqldriver.Use()
	pgdriver.Use()
	app := kago.New() // router + admin + database + shell
	app.Run()
}
```

### BareBone Router (no assets cloned)
```go
package main

import (
	"github.com/kamalshkeir/kago"
)

func main() {
	app := kago.BareBone() // router only, bot need for sql drivers
	app.Run()
}
```

#### 1- running 'go run main.go' the first time, will clone assets folder if run with kago.New and will not if run using kago.BareBone
```shell
go run main.go
```

#### 2- create super user
```shell
go run main.go shell    

-> createsuperuser
```

#### 3- Run the server:
```shell
# run localy:
go run main.go # default: -h localhost -p 9313
```
## That's it, you can visit /admin

--- 

# Run HTTPS letsencrypt in production using env vars and tags

##### if you have already certificates

```sh
go run main.go -h example.com -p 443 --cert cerkey.pem --key privkey.pem
```

##### Autocerts (certs generated and renewed automatically) :

```sh
go run main.go -h example.com -p 443 
# this will generate 2 certificates from letsencrypt example.com and www.example.com to directory ./certs
```
##### to add more domains, you can use tag 'domains':

```sh
go run main.go -h example.com -p 443 -domains a.example.com, b.example.com,... 
```


```zsh
# most important tags and env vars that you can override:
HOST         -h            DEFAULT: "localhost"
PORT         -p			   DEFAULT: "9313"
DOMAINS      -domains	   DEFAULT: ""
CERT 	     -cert         DEFAULT: ""
KEY 	     -key          DEFAULT: ""
PROFILER     -profiler     DEFAULT: false
DOCS         -docs         DEFAULT: false
LOGS         -logs         DEFAULT: false
MONITORING   -monitoring   DEFAULT: false
```



---


### Watcher or Auto-reloader 
### install
```shell
go install github.com/kamalshkeir/kago/cmd/katcher
```
### or get the binary from releases
Then you can run:
```shell
katcher --root ${PWD} (will watch all files at root)
katcher --root ${PWD} --watch assets/templates,assets/static (will watch only '${PWD}/assets/templates' and '${PWD}/assets/static' folders)
```

---

# Generated Admin Dashboard
##### an easy way to create your own theme, is to modify files inside assets , upload the assets folder into a repo and set these 2 values:
```shell
settings.REPO_USER="YOUR_REPO_USER" // default kamalshkeir
settings.REPO_NAME="YOUR_REPO_NAME" // default kago-assets
```

##### you can easily override any handler of any url by creating a new one with the same method and path.<br> for example, to override the handler at GET /admin/login:
```go
app.GET("/admin/login",func(c *kamux.Context) {
	...
})
// this will add another handler, and remove the old one completely 
// so it is very safe to do it this way also
```

### Admin + PWA default handlers
```go
// all these handlers can be overriden
r.GET("/mon/ping",func(c *kamux.Context) {c.Status(200).Text("pong")})
r.GET("/offline",OfflineView) 
r.GET("/manifest.webmanifest",ManifestView) 
r.GET("/sw.js",ServiceWorkerView) 
r.GET("/robots.txt",RobotsTxtView) 
r.GET("/admin", kamux.Admin(IndexView))
r.GET("/admin/login",kamux.Auth(LoginView))
r.POST("/admin/login",kamux.Auth(LoginPOSTView))
r.GET("/admin/logout", LogoutView)
r.POST("/admin/delete/row", kamux.Admin(DeleteRowPost))
r.POST("/admin/update/row", kamux.Admin(UpdateRowPost))
r.POST("/admin/create/row", kamux.Admin(CreateModelView))
r.POST("/admin/drop/table", kamux.Admin(DropTablePost))
r.GET("/admin/table/model:str", kamux.Admin(AllModelsGet))
r.POST("/admin/table/model:str", kamux.Admin(AllModelsPost))
r.GET("/admin/get/model:str/id:int", kamux.Admin(SingleModelGet))
r.GET("/admin/export/table:str", kamux.Admin(ExportView))
r.POST("/admin/import", kamux.Admin(ImportView))
r.GET("/logs",kamux.Admin(LogsGetView))
r.SSE("/sse/logs",kamux.Admin(LogsSSEView))

// Example : how to override a handler
// add it before kago.New()
admin.LoginView=func(c *kamux.Context) {
	...		
}
...


// Example : how to override a handler middleware
kamux.Auth = func(handler kamux.Handler) kamux.Handler { // handlerFunc
		...
}
// Example : how to override a Global middleware
kamux.GZIP = func(handler http.Handler) http.Handler { // Handler
		...
}
```
### Admin Search
<div align="center">
	<img src="https://user-images.githubusercontent.com/54605903/193479905-3dcbd6fd-2af6-4169-b409-a8e9dd940994.png" width="auto" height="auto">
</div>

---
# Routing
### Using GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS, HandlerFunc

```go
package main

import (
	"github.com/kamalshkeir/kago"
	"github.com/kamalshkeir/kago/core/kamux"
)


func main() {
    // Creates a KaGo router
	app := kago.New()

    // You can add GLOBAL middlewares easily  (GZIP,CSRF,LIMITER,RECOVERY)
	app.UseMiddlewares(kamux.GZIP)

	// this is an adapter for go std http handlerfunc, example of getting param param1 of type int (regex validated)
	app.HandlerFunc("*","/test/param1:int/*",func(w http.ResponseWriter, r *http.Request) {
		params,ok := kamux.ParamsHandleFunc(r)
		if ok {
			...
		}
		w.Write([]byte("ok"))
	})
	app.HandlerFunc("GET","/",func(w http.ResponseWriter, r *http.Request) { // using net/http handlerFunc
		w.Write([]byte("hello world"))
	})

	// handle any method kamux Handler 
	app.Handle("*","/any/:test",func(c *kamux.Context) {
		logger.Success(c.Params["test"])
		c.Json(kamux.M{
			"message":"message here",
		})
	})
	app.Handle("GET","/any2/:test",func(c *kamux.Context) {
		logger.Success(c.Params)
		c.Json(kamux.M{
			"message":"message here",
		})
	})

	// handle GET /any/hello and /any/world
	app.GET("/any/(hello|world)",func(c *kamux.Context) { 
		c.Text("hello world")
	})

	// middlewares for single handler (Auth,Admin,BasicAuth)
	// Auth check if user is authenticated and pass to c.Html '.User' and '.Request' accessible in all templates
	app.GET("/",kamux.Auth(IndexHandler))
	app.POST("/somePost", posting)
	app.PUT("/somePut", putting)
	app.PATCH("/somePatch", patching)
	app.DELETE("/someDelete", deleting)
	app.HEAD("/someDelete", head)
	app.OPTIONS("/someDelete", options)


	app.Run()
}

var IndexHandler = func(c *kamux.Context) {
	if param1,ok := c.Params["param1"];ok {
		c.Json(kamux.M{ // Status default to 200, so no need to add it
			"param1":param1,
		}) // send json
	} else {
		c.Status(400).Json(kamux.M{
			"error":"message",
		}) // send json with status code 400
	}
}
```

## Websockets + Server Sent Events
### After this one, you will stop being afraid to try websockets everywhere

```go
func main() {
	app := kago.New()
	
	// no need to upgrade the request , all you need to worry about is
	// inside this handler, you can enjoy realtime communication
	app.WS("/ws/test",func(c *kamux.WsContext) {
		rand := utils.GenerateRandomString(5)
		c.AddClient(rand) // add connection to broadcast list

		// listen for messages coming from 1 user
		for {
			// receive Json
			mapStringAny,err := c.ReceiveJson()
			// receive Text
			str,err := c.ReceiveText()
			if err != nil {
				// on error you can remove client from broadcastList and break the loop
				c.RemoveRequester(rand)
				break
			}

			// send Json to current user
			err = c.Json(kamux.M{
				"Hello":"World",
			})

			// send Text to current user
			err = c.Text("any data string")

			// broadcast to all connected users
			c.Broadcast(kamux.M{
				"you can send":"struct insetead of maps here",
			})

			// broadcast to all connected users except current user, the one who send the last message
			c.BroadcastExceptCaller(map[string]any{
				"you can send":"struct insetead of maps here",
			})

		}
	})
	
	app.Run()
}
```
### Server Sent Events
```go
func main() {
	app := kago.New()
	
	// will be hitted every 1-2 sec, you can check anything if change and send data on the fly using c.StreamResponse
	app.SSE("/sse/logs",func(c *kamux.Context) {
		lenStream := len(logger.StreamLogs)
		if lenStream > 0 {
			lastOne := lenStream-1
			err := c.StreamResponse(logger.StreamLogs[lastOne])
			if err == nil{
				logger.StreamLogs=[]string{}
			}
		}
	})
	
	app.Run()
}
```
---

# Parameters (path + query)

```go
func main() {
	app := kago.New()

    // Query string parameters are parsed using the existing underlying request object
    // request url : /?page=3
    app.GET("/",func(c *kamux.Context) {
		page := c.QueryParam("page")
		if page != "" {
			c.Json(kamux.M{
				"page":page,
			})
		} else {
			c.Status(404)
		}
	})

    // This handler will match /anyString but will not match /
    // accepted param Type: string,slug,int,float and validated on the go, before it hit the handler
    app.POST("/param1:slug",func(c *kamux.Context) {
		if param1,ok := c.Params["param1"];ok {
			c.Json(kamux.M{
				"param1":param1,
			})
		} else {
			c.Status(404).Text("Not Found")
		}
	})

	// OR 

	// param1 can be ascii, no symbole
	app.PATCH("/test/:param1",func(c *kamux.Context) {
		if param1,ok := c.Params["param1"];ok {
			c.Json(kamux.M{
				"param1":param1,
			})
		} else {
			c.Status(404).Text("Not Found")
		}
	})

	// param1 can be ascii, no symbole -> /test/anything will work
	app.POST("/test/param1:str",func(c *kamux.Context) 

	//  /test/anything will not work, should be int, validated with regex -> /100 will work
	app.POST("/test/param1:int",func(c *kamux.Context) 

	//  2 digit float -> /test/3.45 will work
	app.POST("/test/param1:float",func(c *kamux.Context)

	// slug param should match abcd-efgh, no spacing 
	app.POST("/test/param1:slug",func(c *kamux.Context) 


	app.Run()
}
```


# Context Http
###### There is also WsContext seen above

```go
func main() {
	app := kago.New()

    // Query string parameters are parsed using the existing underlying request object
    // request url : /?page=3
    app.GET("/",func(c *kamux.Context) {
		page := c.QueryParam("page")
		if page != "" {
			c.Status(200).Json(map[string]any{
				"page":page,
			})
		} else {
			c.SetStatus(404)
		}
	})

    // This handler will match /anyString but will not match /
    // accepted param Type: string,slug,int,float and validated on the go, before it hit the handler
    app.POST("/param1:slug",func(c *kamux.Context) {
		if param1,ok := c.Params["param1"];ok {
			c.Json(map[string]any{
				"param1":param1,
			})
		} else {
			c.Status(404).Text("Not Found")
		}
	})

	// and many more
	c.AddHeader(key,value string) // append a header if key exist
	c.SetHeader(key,value string) // replace a header if key exist
	c.writeHeader(statusCode int) // set status code like c.writeHeader(statusCode int)
	c.SetStatus(statusCode int) // set status code like c.writeHeader(statusCode int)
	c.IsAuthenticated() bool // return true if valid user authenticated
	c.User() models.User // get User from request if middlewares.Auth or middlewares.Admin used
	c.Status(200).Json(body any)
	c.Status(200).JsonIndent(body any)
	c.Status(200).Html(template_name string, data map[string]any)
	c.Status(301).Redirect(path string) // redirect to path
	c.BodyJson() map[string]any // get request body as map
	c.BodyText() string // get request body as string
	c.StreamResponse(response string) error //SSE
	c.ServeFile("application/json; charset=utf-8", "./test.json")
	c.ServeEmbededFile(content_type string,embed_file []byte)
	c.ParseMultipartForm(size ...int64) (formData url.Values, formFiles map[string][]*multipart.FileHeader) // return form data and form files
	c.UploadFile(received_filename,folder_out string, acceptedFormats ...string) (string,[]byte,error) // UploadFile upload received_filename into folder_out and return url,fileByte,error
	c.UploadFiles(received_filenames []string,folder_out string, acceptedFormats ...string) ([]string,[][]byte,error) // UploadFilse handle also if it's the same name but multiple files or multiple names multiple files
	c.DeleteFile(path string) error
	c.Download(data_bytes []byte, asFilename string)
	c.EnableTranslations() // EnableTranslations get user ip, then location country using nmap, so don't use it if u don't have it install, and then it parse csv file to find the language spoken in this country, to finaly set cookie 'lang' to 'en' or 'fr'... 
	c.GetUserIP() string // get user ip

	app.Run()
}
```

## Multipart/Urlencoded Form

```go
func main() {
	app := kago.New()

	app.POST("/ajax", func(c *kamux.Context) {
        // get json body to map[string]any
		requestData := c.BodyJson()
        if email,ok := requestData["email"]; ok {
            ...
        }

        if err != nil {
            c.Status(400).Json(kamux.M{
			    "error":"User doesn not Exist",
		    })
        }
	})
	app.Run()
}
```

## Upload file
```go
func main() {
	app := kago.New()

	// kamux.MultipartSize = 10<<20 memory limit set to 10Mb
	app.POST("/upload", func(c *kamux.Context) {
		// UploadFileFromFormData upload received_filename into folder_out and return url,fileByte,error
        //c.UploadFile(received_filename,folder_out string, acceptedFormats ...string) (string,[]byte,error)
        pathToFile,dataBytes,err := c.UploadFile("filename_from_form","images","png","jpg")
		// you can save pathToFile in db from here
		c.Status(200).Text(file.Filename + " uploaded")
	})


	app.Run()
}
```

## Cookies
```go
func main() {
	app := kago.New()

	kamux.COOKIE_EXPIRE= time.Now().Add(7 * 24 * time.Hour)
	
	app.POST("/ajax", func(c *kamux.Context) {
        // get json body to map[string]any
		requestData := c.BodyJson()
        if email,ok := requestData["email"]; ok {
            ...
        }

		// set cookie
		c.SetCookie(key,value string)
		c.GetCookie(key string)
		c.DeleteCookie(key string)
	})

	app.Run()
}
```

## HTML functions maps
```go

// To add one, automatically loaded into your html
app.NewFuncMap(funcName string, function any)

/* FUNC MAPS */
var functions = template.FuncMap{
	"contains": func(str string, substrings ...string) bool 
	"startWith": func(str string, substrings ...string) bool 
	"finishWith": func(str string, substrings ...string) bool 
	"generateUUID": func() template.HTML 
	"add": func(a int,b int) int 
	"safe": func(str string) template.HTML 
	"timeFormat":func (t any) string 
	"truncate": func(str any,size int) any 
	"csrf_token":func (r *http.Request) template.HTML // generate input with id csrf_token
	"date": func(t any) string // dd Month yyyy
	"slug": func(str string) string
	"translateFromLang":func (translation,language  string) any 
	"translateFromRequest":func (translation string, request *http.Request) any 
}

```
---
# Add Custom Static And Templates Folders
##### you can build all your static and templates files into the binary by simply embeding folder using app.Embed

```go
app.Embed(staticDir *embed.FS, templateDir *embed.FS)
app.ServeLocalDir(dirPath, webPath string)
app.ServeEmbededDir(pathLocalDir string, embeded embed.FS, webPath string)
app.AddLocalTemplates(pathToDir string) error
app.AddEmbededTemplates(template_embed embed.FS,rootDir string) error

// examples:
// initial static and templates 
//settings.STATIC_DIR="static" --> to serve at /static (optional)
//settings.TEMPLATE_DIR="templates" --> templates inside this folder are used with c.Html

app.GET("/test/id:int",func(c *kamux.Context) {
	id,ok := c.Params["id"]
	if !ok {
		...
		return
	}
	c.Html("index.html",kamux.M{
		"id":id,
	})
})

// you can add static and templates folders
app.ServeLocalDir("/path/to/static","static") // serve local /path/to/static folder at /static/*
app.AddLocalTemplates("/path/to/templates1") // templates inside this folder are used with c.Html
app.AddLocalTemplates("/path/to/templates2")

// you can serve also add embeded templates and static folders
router.ServeEmbededDir("/path/to/static", Static, "static") // serve embeded assets/static folder at endpoint /static/*
router.AddEmbededTemplates(Templates,"/path/to/templates")

```
---
# Middlewares

#### Global server middlewares
```go
func main() {
	app := New()
	app.UseMiddlewares(
		kamux.GZIP,
		kamux.CSRF,
		kamux.LIMITER,
		kamux.RECOVERY,
	)

	// GZIP nothing to do , just add it

	// LOGS /!\no need to add it using app.UseMiddlewares, instead you have a flag --logs that enable /logs
	// when logs middleware used, you will have a colored log for requests and also all logs from logger library displayed in the terminal and at /logs enabled for admin only
	// add the middleware like above and enjoy SSE logs in your browser not persisting if you ask

	// LIMITER 
	// if enabled , requester are blocked 5 minutes if make more then 50 request/s , you can change these values:
	kamux.LIMITER_TOKENS=50
	kamux.LIMITER_TIMEOUT=5*time.Minute

	// RECOVERY
	// will recover any error and log it, you can see it in console and also at /logs if LOGS middleware enabled

	// CORS
	// this is how to use CORS, it's applied globaly , but defined by the handler, all methods except GET of course
	app.AllowOrigines(origines ...string) // allow origines global, can be "*" to allow all
	app.POST(pattern string, handler kamux.Handler, allowed_origines ...string)
	app.POST("/users/post",func(c *kamux.Context) {
		// allow origine for domain.com and domain2.com and same origin
	},"domain.com","domain2.com")

	// CSRF
	// CSRF middleware get csrf_token from header, middleware will put it in cookies
	// this is a helper javascript function you can use to get cookie csrf and the set it globaly for all your request
	function getCookie(name) {
		var cookieValue = null;
		if (document.cookie && document.cookie != '') {
			var cookies = document.cookie.split(';');
			for (var i = 0; i < cookies.length; i++) {
				var cookie = cookies[i].trim();
				// Does this cookie string begin with the name we want?
				if (cookie.substring(0, name.length + 1) == (name + '=')) {
					cookieValue = decodeURIComponent(cookie.substring(name.length + 1));
					break;
				}
			}
		}
		return cookieValue;
	}
	let csrftoken = getCookie("csrf_token");

	// or a you have also a template function called csrf_token
	// you can use it in templates to render a hidden input of csrf_token
	<form id="form1">
		{{ csrf_token .request }}
	</form>
	// you can get it from the input from js and send it in headers
	// middleware csrf will try to find this header name:
	COOKIE NAME: 'csrf_token'
	HEADER NAME: 'X-CSRF-Token'



	app.Run()
}
```

## Handler middlewares

```go
// USAGE:
r.GET("/admin", kamux.Admin(IndexView)) // will check from session cookie if user.is_admin is true
r.GET("/admin/login",kamux.Auth(LoginView)) // will get session from cookies decrypt it and validate it
r.GET("/test",kamux.BasicAuth(LoginView,"username","password"))
```
---

# ORM
###### i waited go1.18 and generics to make this package orm to keep performance at it's best with convenient way to query your data, even from multiple databases
## queries are cached using powerfull eventbus style that empty cache when changes in database may corrupt your data, so use it until you have a problem with it
###### to disable it : 
``` 
orm.UseCache=false
```

#### Let's start by ways to add new database:
```go
orm.NewDatabaseFromDSN(dbType,dbName string,dbDSN ...string) (error)
orm.NewDatabaseFromConnection(dbType,dbName string,conn *sql.DB) (error)
orm.GetConnection(dbName ...string) // GetConnection return default connection for orm.DefaultDatabase (if dbName not given or "" or "default") else it return the specified one
orm.UseForAdmin(dbName string) // UseForAdmin use specific database in admin panel if many
orm.GetDatabases() []DatabaseEntity // GetDatabases get list of databases available to your app
orm.GetDatabase() *DatabaseEntity // GetDatabase return the first connected database orm.DefaultDatabase if dbName "" or "default" else the matched db
```

#### Utility database queries:
```go
orm.GetAllTables(dbName ...string) []string // if dbName not given, .env used instead to default the db to get all tables in the given db
orm.GetAllColumnsTypes(table string, dbName ...string) map[string]string // clear i think
orm.CreateUser(email,password string,isAdmin int, dbName ...string) error // password will be hashed using argon2
```

---

# Migrations
### using the shell, you can migrate a .sql file ```go run main.go shell```

### OR
### you can migrate from a struct using [Auto Migrate](#automigrate-usage)

### when kago app executed, all models registered using AutoMigrate will be synchronized with the database so if you add a field to you struct or add a column to your table, you will have a prompt proposing migration

### execute AutoMigrate and don't think about it, it will handle all synchronisations between your project structs types like in the example Bookmark below

### For instance you can add foreign keys column to your table by adding extra field to your struct and restart your app, you can also remove foreign keys. 

### If you need to change a tag, remove the field, restart, put the new one with the new tag, restart again, that's it 

---
### Available Tags by struct field type and [Examples](#automigrate-usage) :
---

#String Field:
<table>
<tr>
<th>Without parameter&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;</th>
<th>With parameter&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;</th>
</tr>
<tr>
<td>
 
```
*  	text (create column as TEXT not VARCHAR)
*  	notnull
*  	unique
*   iunique // insensitive unique
*  	index, +index, index+ (INDEX ascending)
*  	index-, -index (INDEX descending)
*  	default (DEFAULT '')
```
</td>
<td>

```
* 	default:'any' (DEFAULT 'any')
*	mindex:...
* 	uindex:username,Iemail // CREATE UNIQUE INDEX ON users (username,LOWER(email)) 
	// 	email is lower because of 'I' meaning Insensitive for email
* 	fk:...
* 	size:50  (VARCHAR(50))
* 	check:...
```

</td>
</tr>
</table>


---



# Int, Uint, Int64, Uint64 Fields:
<table>
<tr>
<th>Without parameter&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;</th>
</tr>
<tr>
<td>
 
```
*   -  			 (To Ignore a field)
*   autoinc, pk  (PRIMARY KEY)
*   notnull      (NOT NULL)
*  	index, +index, index+ (CREATE INDEX ON COLUMN)
*  	index-, -index(CREATE INDEX DESC ON COLUMN)     
*   unique 		 (CREATE UNIQUE INDEX ON COLUMN) 
*   default		 (DEFAULT 0)
```
</td>
</tr>

<tr><th>With parameter&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;</th></tr>
<tr>
<td>

```
Available 'on_delete' and 'on_update' options: cascade,(donothing,noaction),(setnull,null),(setdefault,default)

*   fk:{table}.{column}:{on_delete}:{on_update} 
*   check: len(to_check) > 10 ; check: is_used=true (You can chain checks or keep it in the same CHECK separated by AND)
*   mindex: first_name, last_name (CREATE MULTI INDEX ON COLUMN + first_name + last_name)
*   uindex: first_name, last_name (CREATE MULTI UNIQUE INDEX ON COLUMN + first_name + last_name) 
*   default:5 (DEFAULT 5)
```

</td>
</tr>
</table>

---


# Bool : bool is INTEGER NOT NULL checked between 0 and 1 (in order to be consistent accross sql dialects)
<table>
<tr>
<th>Without parameter&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;</th>
<th>With parameter&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;</th>
</tr>
<tr>
<td>
 
```
*  	index, +index, index+ (CREATE INDEX ON COLUMN)
*  	index-, -index(CREATE INDEX DESC ON COLUMN)  
*   default (DEFAULT 0)
```
</td>
<td>

```
*   default:1 (DEFAULT 1)
*   mindex:...
*   fk:...
```

</td>
</tr>
</table>

---

# time.Time :
<table>
<tr>
<th>Without parameter&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;</th>
<th>With parameter</th>
</tr>
<tr>
<td>
 
```
*  	index, +index, index+ (CREATE INDEX ON COLUMN)
*  	index-, -index(CREATE INDEX DESC ON COLUMN)  
*   now (NOT NULL and defaulted to current timestamp)
*   update (NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP)
```
</td>
<td>

```
*   fk:...
*   check:...
```

</td>
</tr>
</table>

---

# Float64 :
<table>
<tr>
<th>Without parameter&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;</th>
<th>With parameter&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;</th>
</tr>
<tr>
<td>
 
```
*   notnull
*  	index, +index, index+ (CREATE INDEX ON COLUMN)
*  	index-, -index(CREATE INDEX DESC ON COLUMN)  
*   unique
*   default
```
</td>
<td>

```
*   default:...
*   fk:...
*   mindex:...
*   uindex:...
*   check:...
```

</td>
</tr>
</table>

---

# AutoMigrate Usage


```go

orm.AutoMigrate[T comparable](tableName string, dbName ...string) error 

//Examples:
// this is the actual user model used initialy
type User struct {
	Id        int       `json:"id,omitempty" orm:"pk"`
	Uuid      string    `json:"uuid,omitempty" orm:"size:40"`
	Email     string    `json:"email,omitempty" orm:"size:50;iunique"`
	Password  string    `json:"password,omitempty" orm:"size:150"`
	IsAdmin   bool      `json:"is_admin,omitempty" orm:"default:false"`
	Image     string    `json:"image,omitempty" orm:"size:100;default:''"`
	CreatedAt time.Time `json:"created_at,omitempty" orm:"now"`
}

type Bookmark struct {
	Id      uint   `orm:"pk"`
	UserId  int    `orm:"fk:users.id:cascade:setnull"` // options cascade,(donothing,noaction),(setnull,null),(setdefault,default)
	IsDone	bool   
	ToCheck string `orm:"size:50; notnull; check: len(to_check) > 2 AND len(to_check) < 10; check: is_done=true"`  // column type will be VARCHAR(50)
	Content string `orm:"text"` // column type will be TEXT, and will have Rich Text Editor in admin panel
	UpdatedAt time.Time `orm:"update"` // will update when model updated, handled by triggers for sqlite, coakroach and postgres, and builtin mysql
	CreatedAt time.Time `orm:"now"` // now is default to current timestamp and of type TEXT for sqlite
}


// TO DEBUG , or to see queries executed on migrations, set orm.DEBUG=true


// To migrate/connect/sync with database:
err := orm.AutoMigrate[Bookmark]("bookmarks")

// after this command if you remove a field from struct model, you will have a prompt wit suggestions to resolve the problem

// will produce:
CREATE TABLE IF NOT EXISTS bookmarks (
  id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, 
  user_id INTEGER, 
  is_used INTEGER NOT NULL CHECK (
    is_used IN (0, 1)
  ), 
  to_check VARCHAR(50) UNIQUE NOT NULL CHECK (
    length(to_check) > 10
  ), 
  content TEXT, 
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, 
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

##### And in the admin panel, you can see that 'bookmarks' table created, with a rich text editor for the column 'content'
##### this is because all columns with field type TEXT will be rendered as a rich text editor automatically
<div align="center">
	<img src="https://user-images.githubusercontent.com/54605903/183331872-85647169-0d48-4bca-a79b-c2f87b61fb13.png" width="auto" style="margin:0 auto 0 auto;"/>
</div>



# ORM Triggers
```go
func AddTrigger(onTable, col, bf_af_UpdateInsertDelete string,ofColumn, stmt string,forEachRow bool,whenEachRow string, dbName ...string)
func DropTrigger(onField,tableName string)

//example:
orm.AddTrigger("users","email","AFTER UPDATE","email","UPDATE users SET uuid='test' where users.id=NEW.id",false,"")
// this trigger will update uuid to 'test' whenever email is updated
```



# Queries and Sql Builder (handle multiple database)
#### to query, insert, update and delete using structs:
```go
orm.Model[T comparable]() *Builder[T] // starter
orm.BuilderS[T comparable]() *Builder[T] // starter
(b *Builder[T]).Database(dbName string) *Builder[T]
(b *Builder[T]).Insert(model *T) (int, error)
(b *Builder[T]).Set(query string, args ...any) (int, error)
(b *Builder[T]).Delete() (int, error) // finisher
(b *Builder[T]).Drop() (int, error) // finisher
(b *Builder[T]).Select(columns ...string) *Builder[T]
(b *Builder[T]).Where(query string, args ...any) *Builder[T]
(b *Builder[T]).Query(query string, args ...any) *Builder[T]
(b *Builder[T]).Limit(limit int) *Builder[T]
(b *Builder[T]).Context(ctx context.Context) *Builder[T]
(b *Builder[T]).Page(pageNumber int) *Builder[T]
(b *Builder[T]).OrderBy(fields ...string) *Builder[T]
(b *Builder[T]).Debug() *Builder[T] // print the query statement
(b *Builder[T]).All() ([]T, error) // finisher
(b *Builder[T]).One() (T, error) // finisher
```
# Examples
```go
// then you can query your data as models.User data
// you have 2 finisher : All and One

// Select and Pagination
orm.Model[models.User]().Select("email","uuid").OrderBy("-id").Limit(PAGINATION_PER).Page(1).All()

// INSERT
uuid,_ := orm.GenerateUUID()
hashedPass,_ := hash.GenerateHash("password")
orm.Model[models.User]().Insert(&models.User{
	Uuid: uuid,
	Email: "test@example.com",
	Password: hashedPass,
	IsAdmin: false,
	Image: "",
	CreatedAt: time.Now(),
})

//if using more than one db
orm.Database[models.User]("dbNameHere").Where("id = ? AND email = ?",1,"test@example.com").All() 

// where
orm.Model[models.User]().Where("id = ? AND email = ?",1,"test@example.com").One() 

// delete
orm.Model[models.User]().Where("id = ? AND email = ?",1,"test@example.com").Delete()

// drop table
orm.Model[models.User]().Drop()

// update
orm.Model[models.User]().Where("id = ?",1).Set("email = ?","new@example.com")

```
#### to query, insert, update and delete using map[string]any:
```go
orm.Exec(dbName,query string, args ...any) error
orm.Query(dbName,query string, args...)  ([]map[string]any,error)
orm.Table(tableName string) *BuilderM // starter
orm.BuilderMap(tableName string) *BuilderM // starter
(b *BuilderM).Database(dbName string) *BuilderM
(b *BuilderM).Select(columns ...string) *BuilderM
(b *BuilderM).Where(query string, args ...any) *BuilderM
(b *BuilderM).Query(query string, args ...any) *BuilderM
(b *BuilderM).Limit(limit int) *BuilderM
(b *BuilderM).Page(pageNumber int) *BuilderM
(b *BuilderM).OrderBy(fields ...string) *BuilderM
(b *BuilderM).Context(ctx context.Context) *BuilderM
(b *BuilderM).Debug() *BuilderM
(b *BuilderM).All() ([]map[string]any, error)
(b *BuilderM).One() (map[string]any, error)
(b *BuilderM).Insert(fields_comma_separated string, fields_values []any) (int, error)
(b *BuilderM).Set(query string, args ...any) (int, error)
(b *BuilderM).Delete() (int, error)
(b *BuilderM).Drop() (int, error)
```
# Examples
```go
// for using maps , no need to link any model of course
// then you can query your data as models.User data
// you have 2 finisher : All and One for queries

// Select and Pagination
orm.Table("users").Select("email","uuid").OrderBy("-id").Limit(PAGINATION_PER).Page(1).All()

// INSERT
uuid,_ := orm.GenerateUUID()
hashedPass,_ := hash.GenerateHash("password")

orm.Table("users").Insert(
	"uuid,email,password,is_admin,created_at",
	uuid,
	"email@example.com",
	hashedPass,
	false,
	time.Now()
)

//if using more than one db
orm.Database("dbNameHere").Where("id = ? AND email = ?",1,"test@example.com").All() 

// where
orm.Table("users").Where("id = ? AND email = ?",1,"test@example.com").One() 

// delete
orm.Table("users").Where("id = ? AND email = ?",1,"test@example.com").Delete()

// drop table
orm.Table("users").Drop()

// update
orm.Table("users").Where("id = ?",1).Set("email = ?","new@example.com")

```

---

# SHELL
##### Very useful shell to explore, no need to install extra dependecies or binary, you can run:
```shell
go run main.go shell
go run main.go help
```

```shell
AVAILABLE COMMANDS:
[databases, use, tables, columns, migrate, createsuperuser, 
createuser, getall, get, drop, delete, clear/cls, q/quit/exit, help/commands]
  'databases':
	  list all connected databases

  'use':
	  use a specific database

  'tables':
	  list all tables in database

  'columns':
	  list all columns of a table

  'migrate':
	  migrate initial users to env database

  'createsuperuser':
	  create a admin user

  'createuser':
	  create a regular user

  'getall':
	  get all rows given a table name

  'get':
	  get single row wher field equal_to

  'delete':
	  delete rows where field equal_to

  'drop':
	  drop a table given table name

  'clear/cls':
	  clear console
```
---
# Env Loader
#### this minimalistic package is one of my favorite, you basicaly need to Load env variables from file and to fill a struct Config with these values
#### First you may need to env vars from file and set them using : ``` envloader.Load(...files)```
###### here is how tediously i was loading env variables:
## Before:
```go
//  
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
				settings.Config.EmbedStatic=b
			}
		case "EMBED_TEMPLATES":
			if b,err := strconv.ParseBool(v);!logger.CheckError(err) {
				settings.Config.EmbedTemplates=b
			}
		case "DB_TYPE":
			if v == "" {v="sqlite"}
			settings.Config.Db.Type=v
		case "DB_DSN":
			if v == "" {v="db.sqlite"}
			settings.Config.DbDSN=v
		case "DB_NAME":
			if v == "" {
				logger.Error("DB_NAME from env file cannot be empty")
				os.Exit(1)
			}
			settings.Config.DbName=v
		case "SMTP_EMAIL":
			settings.Config.SmtpEmail=v
		case "SMTP_PASS":
			settings.Config.SmtpPass=v
		case "SMTP_HOST":
			settings.Config.SmtpHost=v
		case "SMTP_PORT":
			settings.Config.SmtpPort=v
		}
	}
}
```
## After:
```go
envloader.Load(".env") // load env files and add to env vars
// the command:
err := envloader.FillStruct(&Config) // fill struct with env vars
```

## Struct to fill
```go
type GlobalConfig struct {
	Host       string `env:"HOST|localhost"` // DEFAULTED: if HOST not found default to localhost
	Port       string `env:"PORT|9313"`
	Embed struct {
		Static    bool `env:"EMBED_STATIC|false"`
		Templates bool `env:"EMBED_TEMPLATES|false"`
	}
	Db struct {
		Name     string `env:"DB_NAME|db"`
		Type     string `env:"DB_TYPE"` // REEQUIRED: this env var is required, you will have error if empty
		DSN      string `env:"DB_DSN|"` // NOT REQUIRED: if DB_DSN not found it's not required, it's ok to stay empty
	}
	Smtp struct {
		Email string `env:"SMTP_EMAIL|"`
		Pass  string `env:"SMTP_PASS|"`
		Host  string `env:"SMTP_HOST|"`
		Port  string `env:"SMTP_PORT|"`
	}
	Profiler   bool   `env:"PROFILER|false"`
	Docs       bool   `env:"DOCS|false"`
	Logs       bool   `env:"LOGS|false"`
	Monitoring bool   `env:"MONITORING|false"`
}
```
---

# OpenApi documentation ready if enabled using flags or settings vars
```bash
// running the app using flag --docs
go run main.go --docs
// go to /docs
```
## to edit the documentation, you have a docs package 
```go

doc := docs.New()

doc.AddModel("User",docs.Model{
	Type: "object",
	RequiredFields: []string{"email","password"},
	Properties: map[string]docs.Property{
		"email":{
			Required: true,
			Type: "string",
			Example: "example@xyz.com",
		},
		"password":{
			Required: true,
			Type: "string",
			Example: "************",
			Format: "password",
		},
	},
})
doc.AddPath("/admin/login","post",docs.Path{
	Tags: []string{"Auth"},
	Summary: "login post request",
	OperationId: "login-post",
	Description: "Login Post Request",
	Requestbody: docs.RequestBody{
		Description: "email and password for login",
		Required: true,
		Content: map[string]docs.ContentType{
			"application/json":{
				Schema: docs.Schema{
					Ref: "#/components/schemas/User",
				},
			},
		},
	},
	Responses: map[string]docs.Response{
		"404":{Description: "NOT FOUND"},
		"403":{Description: "WRONG PASSWORD"},
		"500":{Description: "INTERNAL SERVER ERROR"},
		"200":{Description: "OK"},
	},
	Consumes: []string{"application/json"},
	Produces: []string{"application/json"},
})
doc.Save()
```
---

# Encryption
```go
// AES Encrypt use SECRET in .env
encryptor.Encrypt(data string) (string,error)
encryptor.Decrypt(data string) (string,error)
```
---

# Hashing
```go
// Argon2 hashing
hash.GenerateHash(password string) (string, error)
hash.ComparePasswordToHash(password, hash string) (bool, error)
```

---

# Eventbus Internal handle any data using generics, just make sure you are using same type for the same topic
```go
eventbus.Subscribe("any_topic",func(data map[string]any) {
	...
})

eventbus.Subscribe("any_topic222",func(data int) {
	...
})

eventbus.Publish("any_topic", map[string]any{
	"type":     "update",
	"table":    b.tableName,
	"database": b.database,
})

eventbus.Publish("any_topic222", 222)
```

---

# LOGS
```sh
go run main.go --logs

will enable:
	- /logs
```
---

# PPROF official golang profiling tools
```sh
go run main.go --profiler

will enable:
	- /debug/pprof
	- /debug/pprof/profile
	- /debug/pprof/heap
	- /debug/pprof/trace
```

---

# Grafana with Prometheus monitoring
### Enable /metrics for prometheus
```sh
go run main.go --monitoring
```

### Create file 'prometheus.yml' anywhere
```yml
scrape_configs:
- job_name: api-server
  scrape_interval: 5s
  static_configs:
  - targets: ['host.docker.internal:9313'] #replace host.docker.internal per localhost if you run it localy
```

### Prometheus
```sh
docker run -d --name prometheus -v {path_to_prometheus.yml}:/etc/prometheus/prometheus.yml -p 9090:9090 prom/prometheus
```
### Grafana
```sh
docker run -d --name grafana -p 3000:3000 grafana/grafana-enterprise
```
```sh
docker exec -it grafana grafana-cli admin reset-admin-password newpass
```
##### Visit   http://localhost:3000 username: admin
### Add data source http://host.docker.internal:9090
## That's it, you can import your favorite dashboard:
 - 10826 
 - 240



---

# Build single binary with all static and html files
```go
//go:embed assets/static
var Static embed.FS
//go:embed assets/templates
var Templates embed.FS

func main() {
	app := New()
	app.UseMiddlewares(middlewares.GZIP)
	app.Embed(&Static,&Templates)
	app.Run()
}

```

# Then
```bash
go build
```

---

# Some Benchmarks
```shell
////////////////////////////////////// postgres without cache
BenchmarkGetAllS-4                  1472            723428 ns/op            5271 B/op         80 allocs/op
BenchmarkGetAllM-4                  1502            716418 ns/op            4912 B/op         85 allocs/op
BenchmarkGetRowS-4                   826           1474674 ns/op            2288 B/op         44 allocs/op
BenchmarkGetRowM-4                   848           1392919 ns/op            2216 B/op         44 allocs/op
BenchmarkGetAllTables-4             1176            940142 ns/op             592 B/op         20 allocs/op
BenchmarkGetAllColumnsTypes-4             417           2862546 ns/op            1456 B/op         46 allocs/op
////////////////////////////////////// postgres with cache
BenchmarkGetAllS-4               2825896               427.9 ns/op           208 B/op          2 allocs/op
BenchmarkGetAllM-4               6209617               188.9 ns/op            16 B/op          1 allocs/op
BenchmarkGetRowS-4               2191544               528.1 ns/op           240 B/op          4 allocs/op
BenchmarkGetRowM-4               3799377               305.5 ns/op            48 B/op          3 allocs/op
BenchmarkGetAllTables-4         76298504                21.41 ns/op            0 B/op          0 allocs/op
BenchmarkGetAllColumnsTypes-4        59004012                19.92 ns/op            0 B/op          0 allocs/op
///////////////////////////////////// mysql without cache
BenchmarkGetAllS-4                  1221            865469 ns/op            7152 B/op        162 allocs/op
BenchmarkGetAllM-4                  1484            843395 ns/op            8272 B/op        215 allocs/op
BenchmarkGetRowS-4                   427           3539007 ns/op            2368 B/op         48 allocs/op
BenchmarkGetRowM-4                   267           4481279 ns/op            2512 B/op         54 allocs/op
BenchmarkGetAllTables-4              771           1700035 ns/op             832 B/op         26 allocs/op
BenchmarkGetAllColumnsTypes-4             760           1537301 ns/op            1392 B/op         44 allocs/op
///////////////////////////////////// mysql with cache
BenchmarkGetAllS-4               2933072               414.5 ns/op           208 B/op          2 allocs/op
BenchmarkGetAllM-4               6704588               180.4 ns/op            16 B/op          1 allocs/op
BenchmarkGetRowS-4               2136634               545.4 ns/op           240 B/op          4 allocs/op
BenchmarkGetRowM-4               4111814               292.6 ns/op            48 B/op          3 allocs/op
BenchmarkGetAllTables-4         58835394                21.52 ns/op            0 B/op          0 allocs/op
BenchmarkGetAllColumnsTypes-4        59059225                19.99 ns/op            0 B/op          0 allocs/op
///////////////////////////////////// sqlite without cache
BenchmarkGetAllS-4                 13664             85506 ns/op            2056 B/op         62 allocs/op
BenchmarkGetAllS_GORM-4            10000            101665 ns/op            9547 B/op        155 allocs/op
BenchmarkGetAllM-4                 13747             83989 ns/op            1912 B/op         61 allocs/op
BenchmarkGetAllM_GORM-4            10000            107810 ns/op            8387 B/op        237 allocs/op
BenchmarkGetRowS-4                 12702             91958 ns/op            2192 B/op         67 allocs/op
BenchmarkGetRowM-4                 13256             89095 ns/op            2048 B/op         66 allocs/op
BenchmarkGetAllTables-4            14264             83939 ns/op             672 B/op         32 allocs/op
BenchmarkGetAllColumnsTypes-4           15236             79498 ns/op            1760 B/op         99 allocs/op
///////////////////////////////////// sqlite with cache
BenchmarkGetAllS-4               2951642               399.5 ns/op           208 B/op          2 allocs/op
BenchmarkGetAllM-4               6537204               177.2 ns/op            16 B/op          1 allocs/op
BenchmarkGetRowS-4               2248524               531.4 ns/op           240 B/op          4 allocs/op
BenchmarkGetRowM-4               4084453               287.9 ns/op            48 B/op          3 allocs/op
BenchmarkGetAllTables-4         52592826                20.39 ns/op            0 B/op          0 allocs/op
BenchmarkGetAllColumnsTypes-4        64293176                20.87 ns/op            0 B/op          0 allocs/op

```
---

# 🔗 Links
[![portfolio](https://img.shields.io/badge/my_portfolio-000?style=for-the-badge&logo=ko-fi&logoColor=white)](https://kamalshkeir.dev/)
[![linkedin](https://img.shields.io/badge/linkedin-0A66C2?style=for-the-badge&logo=linkedin&logoColor=white)](https://www.linkedin.com/in/kamal-shkeir/)


---

# Licence
Licence [BSD-3](./LICENSE)
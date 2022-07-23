# KaGo Web Framework

KaGo is an Django inspired web framework, very intuitive and just like Typescript Frameworks and library everywhere, blazingly fast ^^

If you need performance and good productivity, you will love KaGo.




## Installation

To install KaGo package, you need to install Go and set your Go workspace first.

1. You first need [Go](https://golang.org/) installed (**version 1.18+ is required**), then you can use the below Go command to install KaGo.

```sh
$ go get github.com/kamalshkeir/kago
```

2. Import it in your code:

```go
import "github.com/kamalshkeir/kago"
```

## Quick start

Create main.go:
```go
package main

import "github.com/kamalshkeir/kago"

func main() {
	app := kago.New()
	app.Run()
}
```

```
# running 'go run main.go' will generate assets folder with all static and template files for admin
$ go run main.go

# make sure you copy 'assets/.env.example' beside your main at the root folder 
and rename it to '.env'

# you can change port and host by putting Env Vars 'HOST' and 'PORT' or using flags:
$ go run main.go -h 0.0.0.0 -p 3333 to run on http://0.0.0.0:3333
 
```

## Routing
### Using GET, POST, PUT, PATCH, DELETE

```go
package main

import (
	"github.com/kamalshkeir/kago"
	"github.com/kamalshkeir/kago/core/kamux"
	"github.com/kamalshkeir/kago/core/middlewares"
)


func main() {
    // Creates a KaGo router
	app := kago.New()

    // You can add GLOBAL middlewares easily  (LOGS,GZIP,CORS,CSRF,LIMITER,RECOVERY)
	app.UseMiddlewares(middlewares.GZIP)

	// OR middleware for single handler (Auth,Admin,BasicAuth)
	// Auth ensure user is authenticated and pass to c.HTML '.user' and '.request', so accessible in all templates
	app.GET("/",middlewares.Auth(IndexHandler))

	app.POST("/somePost", posting)
	app.PUT("/somePut", putting)
	app.DELETE("/someDelete", deleting)
	app.PATCH("/somePatch", patching)
	
	app.Run()
}

var IndexHandler = func(c *kamux.Context) {
    if param1,ok := c.Params["param1"];ok {
        c.STATUS(200).JSON(map[string]any{
            "param1":param1,
        }) // send json
    } else {
		// P.S: 
        c.STATUS(404).TEXT("Not Found") // STATUS will not write status to header,only when chained with JSON or TEXT will be executed
		c.WriteHeader(404) // will set the status header
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
			err = c.JSON(map[string]any{
				"Hello":"World",
			})

			// send Text to current user
			err = c.TEXT("any data string")

			// broadcast to all connected users
			c.BROADCAST(map[string]any{
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

## Parameters (path + query)

```go
func main() {
	app := kago.New()

    // Query string parameters are parsed using the existing underlying request object
    // request url : /?page=3
    app.GET("/",func(c *kamux.Context) {
		page := c.QueryParam("page")
		if page != "" {
			c.StatusJSON(200,map[string]any{
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
			c.JSON(200,map[string]any{
				"param1":param1,
			})
		} else {
			c.TEXT(404,"Not Found")
		}
	})

	// OR 

	// param1 can be ascii, no symbole
	app.PATCH("/test/:param1",func(c *kamux.Context) {
		if param1,ok := c.Params["param1"];ok {
			c.JSON(200,map[string]any{
				"param1":param1,
			})
		} else {
			c.TEXT(404,"Not Found")
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


### Router Http Context
###### There is also WsContext seen above

```go
func main() {
	app := kago.New()

    // Query string parameters are parsed using the existing underlying request object
    // request url : /?page=3
    app.GET("/",func(c *kamux.Context) {
		page := c.QueryParam("page")
		if page != "" {
			c.Status(200).JSON(map[string]any{
				"page":page,
			})
		} else {
			c.writeHeader(404)
		}
	})

    // This handler will match /anyString but will not match /
    // accepted param Type: string,slug,int,float and validated on the go, before it hit the handler
    app.POST("/param1:slug",func(c *kamux.Context) {
		if param1,ok := c.Params["param1"];ok {
			c.JSON(200,map[string]any{
				"param1":param1,
			})
		} else {
			c.STATUS(404).TEXT("Not Found")
		}
	})

	// and many more
	c.STATUS(200).JSONIndent(code int, body any)
	c.STATUS(200).HTML(template_name string, data map[string]any)
	c.STATUS(301).REDIRECT(path string) // redirect to path
	c.BODY() map[string]any // get request body
	c.StreamResponse(response string) error //SSE
	c.ServeFile("application/json; charset=utf-8", "./test.json")
	c.ServeEmbededFile(content_type string,embed_file []byte)
	c.UploadFile(received_filename,folder_out string, acceptedFormats ...string) (string,[]byte,error) // UploadFile upload received_filename into folder_out and return url,fileByte,error
	c.DeleteFile(path string) error
	c.Download(data_bytes []byte, asFilename string)
	c.EnableTranslations() // EnableTranslations get user ip, then location country using nmap, so don't use it if u don't have it install, and then it parse csv file to find the language spoken in this country, to finaly set cookie 'lang' to 'en' or 'fr'... 
	c.GetUserIP() string // get user ip

	app.Run()
}
```

### Multipart/Urlencoded Form

```go
func main() {
	app := kago.New()

	app.POST("/ajax", func(c *kamux.Context) {
        // get json body to map[string]any
		requestData := c.BODY()
        if email,ok := requestData["email"]; ok {
            ...
        }

        if err != nil {
            c.STATUS.JSON(map[string]any{
			    "error":"User doesn not Exist",
		    })
        }
	})
	app.Run()
}
```

### Upload file
```go
func main() {
	app := kago.New()

	// kamux.MultipartSize = 10<<20 memory limit set to 10Mb
	app.POST("/upload", func(c *kamux.Context) {
		// UploadFileFromFormData upload received_filename into folder_out and return url,fileByte,error
        //c.UploadFile(received_filename,folder_out string, acceptedFormats ...string) (string,[]byte,error)
        pathToFile,dataBytes,err := c.UploadFile("filename_from_form","images","png","jpg")
		// you can save pathToFile in db from here
		c.STATUS(200).TEXT(file.Filename + " uploaded")
	})


	app.Run(":8080")
}
```

### Cookies
```go
func main() {
	app := kago.New()

	kamux.COOKIE_EXPIRE= time.Now().Add(7 * 24 * time.Hour)
	
	app.POST("/ajax", func(c *kamux.Context) {
        // get json body to map[string]any
		requestData := c.BODY()
        if email,ok := requestData["email"]; ok {
            ...
        }

		// set cookie
		c.SetCookie(key,value string)
		c.GetCookie(key string)
		c.DeleteCookie(key string)
	})

	app.Run(":8080")
}
```

### HTML functions maps
```go

// To add one, automatically loaded into your html
app.NewFuncMap(funcName string, function any)

/* FUNC MAPS */
var functions = template.FuncMap{
	"isBool": func(something any) bool {
	"isTrue": func(something any) bool {
	"contains": func(str string, substrings ...string) bool {
	"startWith": func(str string, substrings ...string) bool {
	"finishWith": func(str string, substrings ...string) bool {
	"generateUUID": func() template.HTML {
	"add": func(a int,b int) int {
	"safe": func(str string) template.HTML {
	"timeFormat":func (t any) string {
	"truncate": func(str any,size int) any {
	"csrf_token":func (r *http.Request) template.HTML {
	"translateFromRequest":func (translation string, request *http.Request) any {
	"translateFromLang":func (translation,language  string) any {
}

```

## Add Custom Static And Templates Folders
##### you can build all your static and templates files into the binary by simply embeding folder using app.Embed

```go
app.Embed(staticDir *embed.FS, templateDir *embed.FS)
app.ServeLocalDir(dirPath, webPath string)
app.ServeEmbededDir(pathLocalDir string, embeded embed.FS, webPath string)
app.AddLocalTemplates(pathToDir string) error
app.AddEmbededTemplates(template_embed embed.FS,rootDir string) error
```

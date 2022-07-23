# KaGo Web Framework

KaGo is an Django inspired web framework, very intuitive and blazingly fast 😅

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
# run main.go, this will generate assets folder with all static and template files for admin
$ go run main.go

# make sure you copy 'assets/.env.example' beside your main on the root folder 
and rename it to '.env'

# you can cange port and host by putting Env Vars 'HOST' and 'PORT' or:
$ go run main.go -h 0.0.0.0 -p 3333 to run on http://0.0.0.0:3333
 
```

## API Examples

You can find a number of ready-to-run examples at [KaGo examples repository](https://github.com/KaGo-gonic/examples).

### Using GET, POST, PUT, PATCH, DELETE

```go
package main

import (
	"github.com/kamalshkeir/kago"
	"github.com/kamalshkeir/kago/core/kamux"
	"github.com/kamalshkeir/kago/core/middlewares"
)

var IndexHandler = func(c *kamux.Context) {
    if param1,ok := c.Params["param1"];ok {
        c.JSON(200,map[string]any{
            "param1":param1,
        })
    } else {
        c.TEXT(404,"Not Found")
    }
}

func main() {
    // Creates a KaGo router
	app := kago.New()

    // You can add GLOBAL middlewares easily  (LOGS,GZIP,CORS,CSRF,Limiter,Recovery)
	app.UseMiddlewares(middlewares.GZIP)

	// OR middleware for single handler (Auth,Admin,BasicAuth)
	app.GET("/",middlewares.Auth(IndexHandler))

	app.POST("/somePost", posting)
	app.PUT("/somePut", putting)
	app.DELETE("/someDelete", deleting)
	app.PATCH("/somePatch", patching)
	
	app.Run()
}
```

### Router Parameters (path + query)

```go
func main() {
	app := kago.New()

    // Query string parameters are parsed using the existing underlying request object
    // request url : /?page=3
    app.GET("/",func(c *kamux.Context) {
		if page := c.QueryParam("page"); page != "" {
			c.JSON(200,map[string]any{
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
			c.JSON(200,map[string]any{
				"param1":param1,
			})
		} else {
			c.TEXT(404,"Not Found")
		}
	})

	app.Run()
}
```

### Multipart/Urlencoded Form

```go
func main() {
	app := kago.New()

	app.POST("/ajax", func(c *kamux.Context) {
        // get json body to map[string]any
		requestData := c.RequestBody()
        if email,ok := requestData["email"]; ok {
            ...
        }

        if err != nil {
            c.Json(http.StatusNotFound, map[string]any{
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
        url,dataBytes,err := c.UploadFile("filename_from_form","images","png","jpg")

		c.TEXT(http.StatusOK, fmt.Sprintf("'%s' uploaded!", file.Filename))
	})


	app.Run(":8080")
}
```


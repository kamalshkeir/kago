# KaGo Web Framework

KaGo is an Django inspired web framework, very intuitive and blazingly fast (i should put it 😅)

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
# run main.go, this will generate assets folder with all static and template files
$ go run main.go

# OR
$ go run main.go -h 0.0.0.0 -p 3333 to run on http://0.0.0.0:3333

# make sure you copy 'assets/.env.example' beside your main on the root folder 
and rename it to '.env'
 
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
        c.Json(200,map[string]any{
            "param1":param1,
        })
    } else {
        c.Text(404,"Not Found")
    }
}

func main() {
    // Creates a KaGo router
	app := kago.New()

    // You can add GLOBAL middlewares easily  (LOGS,GZIP,CORS,CSRF,Limiter,Recovery)
	app.UseMiddlewares(middlewares.GZIP)
    // OR middleware for specifi handler (Auth,Admin,BasicAuth)
	app.Get("/",middlewares.Auth(IndexHandler))

	app.Post("/somePost", posting)
	app.Put("/somePut", putting)
	app.Delete("/someDelete", deleting)
	app.Patch("/somePatch", patching)
	
	app.Run()
}
```

### Router Parameters (path + query)

```go
func main() {
	app := kago.New()

    // Query string parameters are parsed using the existing underlying request object
    // request url : /?page=3
    app.Get("/",func(c *kamux.Context) {
		if page := c.QueryParam("page"); page != "" {
			c.Json(200,map[string]any{
				"page":page,
			})
		} else {
			c.SetStatus(404)
		}
	})

    // This handler will match /anyString but will not match /
    // accepted param Type: string,slug,int,float and validated on the go, before it hit the handler    
    app.Post("/param1:string",func(c *kamux.Context) {
		if param1,ok := c.Params["param1"];ok {
			c.Json(200,map[string]any{
				"param1":param1,
			})
		} else {
			c.Text(404,"Not Found")
		}
	})

	app.Run()
}
```

### Multipart/Urlencoded Form

```go
func main() {
	app := kago.New()

	app.Post("/ajax", func(c *KaGo.Context) {
        // get json body to map[string]any
		requestData := c.GetJson()
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
	router := Kago.New()

	// kamux.MultipartSize = 10<<20 memory limit set to 10Mb
	router.POST("/upload", func(c *KaGo.Context) {
		// UploadFileFromFormData upload received_filename into folder_out and return url,fileByte,error
        //c.UploadFile(received_filename,folder_out string, acceptedFormats ...string) (string,[]byte,error)
        url,dataBytes,err := c.UploadFile("filename_from_form","images","png","jpg")

		c.Text(http.StatusOK, fmt.Sprintf("'%s' uploaded!", file.Filename))
	})


	router.Run(":8080")
}
```


package kamux

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils"
	"github.com/kamalshkeir/kago/core/utils/logger"
)
var MultipartSize = 10<<20

// Context is a wrapper of responseWriter, request, and params map
type Context struct {
	http.ResponseWriter
	*http.Request
	Params map[string]string
	status int
}

func (c *Context) STATUS(code int) *Context {
	c.status=code
	return c
}


// QueryParam get query param
func (c *Context) QueryParam(name string) string {
	return c.Request.URL.Query().Get(name)
}

// JSON return json to the client
func (c *Context) JSON(body any) {
	c.ResponseWriter.Header().Set("Content-Type","application/json")
	if c.status == 0 {c.status=200}
	c.WriteHeader(c.status)
	enc := json.NewEncoder(c.ResponseWriter)
	err := enc.Encode(body)
	logger.CheckError(err)
}

// JSONIndent return json indented to the client
func (c *Context) JSONIndent(code int, body any) {
	c.ResponseWriter.Header().Set("Content-Type","application/json")
	if c.status == 0 {c.status=200}
	c.WriteHeader(c.status)
	enc := json.NewEncoder(c.ResponseWriter)
	enc.SetIndent("","\t")
	err := enc.Encode(body)
	logger.CheckError(err)
}

// TEXT return text with custom code to the client
func (c *Context) TEXT(body string) {
	c.ResponseWriter.Header().Set("Content-Type", "text/plain")
	if c.status == 0 {c.status=200}
	c.WriteHeader(c.status)
	c.ResponseWriter.Write([]byte(body))
}


// HTML return template_name with data to the client
func (c *Context) HTML(template_name string, data map[string]any) {
	const key utils.ContextKey = "user"
	if data == nil { data = make(map[string]any) }
	
	data["request"] = c.Request
	data["logs"] = settings.GlobalConfig.Logs
	user,ok := c.Request.Context().Value(key).(map[string]any)
	if ok {		
		data["is_authenticated"] = true
		data["user"] = user
	} else {
		data["is_authenticated"] = false
		data["user"] = nil
	}
	c.ResponseWriter.Header().Set("Content-Type","text/html; charset=utf-8")
	if c.status == 0 {c.status=200}
	c.WriteHeader(c.status)
	err := allTemplates.ExecuteTemplate(c.ResponseWriter,template_name,data)
	logger.CheckError(err)
}

// RequestBody get json body from request and return map
func (c *Context) RequestBody() map[string]any {
	// USAGE : data := template.GetJson(r)
	d := map[string]any{}
	dec := json.NewDecoder(c.Request.Body)
	defer c.Request.Body.Close()
	if err := dec.Decode(&d); err == io.EOF {
		//empty body
		logger.Error("empty body EOF")
		return nil
	} else if err != nil {
		logger.Error(err)
		return nil
	} else {
		return d
	}
}

// REDIRECT redirect the client to the specified path with a custom code
func (c *Context) REDIRECT(path string) {
	if c.status == 0 {c.status=http.StatusSeeOther}
	http.Redirect(c.ResponseWriter,c.Request,path,c.status)
}

// ServeFile serve a file from handler
func (c *Context) ServeFile(content_type,path_to_file string) {
	c.ResponseWriter.Header().Set("Content-Type", content_type)
	http.ServeFile(c.ResponseWriter, c.Request, path_to_file)
}

// ServeEmbededFile serve an embeded file from handler
func (c *Context) ServeEmbededFile(content_type string,embed_file []byte) {
	c.ResponseWriter.Header().Set("Content-Type", content_type)
		_,_ = c.ResponseWriter.Write(embed_file)
}

// UploadFile upload received_filename into folder_out and return url,fileByte,error
func (c *Context) UploadFile(received_filename,folder_out string, acceptedFormats ...string) (string,[]byte,error) {
	c.Request.ParseMultipartForm(int64(MultipartSize)) //10Mb
	var buff bytes.Buffer
	file, header , err := c.Request.FormFile(received_filename)
	if logger.CheckError(err) {
		return "",nil,err
	}
	defer file.Close()
	// copy the uploaded file to the buffer
	if _, err := io.Copy(&buff, file); err != nil {
		return "",nil,err
	}

	data_string := buff.String()

	// make DIRS if not exist
	err = os.MkdirAll("media/"+folder_out+"/",0664)
	if err != nil {
		return "",nil,err
	}
	// make file
	if len(acceptedFormats) == 0 {
		acceptedFormats=[]string{"jpg","jpeg","png","json"}
	} 
	if utils.StringContains(header.Filename,acceptedFormats...) {
		dst, err := os.Create("media/"+folder_out+"/" + header.Filename)
		if err != nil {
			return "",nil,err
		}
		defer dst.Close()
		dst.Write([]byte(data_string))
		
		url := "media/"+folder_out+"/"+header.Filename
		return url,[]byte(data_string),nil
	} else {
		return "",nil,fmt.Errorf("expecting filename to finish to be %v",acceptedFormats)
	}
}

// DELETE FILE
func (c *Context) DeleteFile(path string) error {
	err := os.Remove("."+path)
	if err != nil {
		return err
	} else {
		return nil
	}
}

// Download download data_bytes(content) asFilename(test.json,data.csv,...) to the client
func (c *Context) Download(data_bytes []byte, asFilename string) {
	bytesReader := bytes.NewReader(data_bytes)
	c.ResponseWriter.Header().Set("Content-Disposition", "attachment; filename=" + strconv.Quote(asFilename))
	c.ResponseWriter.Header().Set("Content-Type", c.Request.Header.Get("Content-Type"))
	io.Copy(c.ResponseWriter,bytesReader)
}

func (c *Context) EnableTranslations() {
	ip := c.GetUserIP()
	if utils.StringContains(ip,"127.0.0.1","localhost","") {
		c.SetCookie("lang","en")
		return
	}
	country := utils.GetIpCountry(ip)
	if country != "" {
		if v,ok := mCountryLanguage.Get(country);ok {
			c.SetCookie("lang",v)
		} 
	} 
}


func (c *Context) GetUserIP() string {
    IPAddress := c.Request.Header.Get("X-Real-Ip")
    if IPAddress == "" {
        IPAddress = c.Request.Header.Get("X-Forwarded-For")
    }
    if IPAddress == "" {
        IPAddress = c.Request.RemoteAddr
    }
    return IPAddress
}
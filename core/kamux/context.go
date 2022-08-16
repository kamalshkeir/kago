package kamux

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/kamalshkeir/kago/core/admin/models"
	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils"
	"github.com/kamalshkeir/kago/core/utils/logger"
)

var MultipartSize = 10 << 20

// Context is a wrapper of responseWriter, request, and params map
type Context struct {
	http.ResponseWriter
	*http.Request
	Params map[string]string
	status int
}

// Status set status to context, will not be writed to header
func (c *Context) Status(code int) *Context {
	c.status = code
	return c
}

// AddHeader Add append a header value to key if exist
func (c *Context) AddHeader(key, value string) {
	c.ResponseWriter.Header().Add(key, value)
}

// SetHeader Set the header value to the new value, old removed
func (c *Context) SetHeader(key, value string) {
	c.ResponseWriter.Header().Set(key, value)
}

// SetHeader Set the header value to the new value, old removed
func (c *Context) SetStatus(statusCode int) {
	c.status = statusCode
	c.WriteHeader(statusCode)
}

// QueryParam get query param
func (c *Context) QueryParam(name string) string {
	return c.Request.URL.Query().Get(name)
}

// Json return json to the client
func (c *Context) Json(body any) {
	c.SetHeader("Content-Type", "application/json")
	if c.status == 0 {
		c.status = 200
	}
	c.WriteHeader(c.status)
	enc := json.NewEncoder(c.ResponseWriter)
	err := enc.Encode(body)
	logger.CheckError(err)
}

// JsonIndent return json indented to the client
func (c *Context) JsonIndent(body any) {
	c.SetHeader("Content-Type", "application/json")
	if c.status == 0 {
		c.status = 200
	}
	c.WriteHeader(c.status)
	enc := json.NewEncoder(c.ResponseWriter)
	enc.SetIndent("", "\t")
	err := enc.Encode(body)
	logger.CheckError(err)
}

// Text return text with custom code to the client
func (c *Context) Text(body string) {
	c.SetHeader("Content-Type", "text/plain")
	if c.status == 0 {
		c.status = 200
	}
	c.WriteHeader(c.status)
	c.ResponseWriter.Write([]byte(body))
}

func (c *Context) IsAuthenticated() bool {
	const key utils.ContextKey = "user"
	if _, ok := c.Request.Context().Value(key).(map[string]any); ok {
		return true
	} else {
		return false
	}
}

func (c *Context) User() models.User {
	const key utils.ContextKey = "user"
	return c.Request.Context().Value(key).(models.User)
}

// Html return template_name with data to the client
func (c *Context) Html(template_name string, data map[string]any) {
	var buff bytes.Buffer
	const key utils.ContextKey = "user"
	if data == nil {
		data = make(map[string]any)
	}

	data["Request"] = c.Request
	data["Logs"] = settings.Config.Logs
	user, ok := c.Request.Context().Value(key).(models.User)
	if ok {
		data["IsAuthenticated"] = true
		data["User"] = user
	} else {
		data["IsAuthenticated"] = false
		data["User"] = nil
	}
	
	err := allTemplates.ExecuteTemplate(&buff, template_name, data)
	if logger.CheckError(err) {
		c.status=http.StatusInternalServerError
		http.Error(c.ResponseWriter,"could not render "+template_name,c.status)
		return
	}

	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	if c.status == 0 {
		c.status = 200
	}
	c.WriteHeader(c.status)

	_,err = buff.WriteTo(c.ResponseWriter)
	logger.CheckError(err)
}

// StreamResponse send SSE Streaming Response
func (c *Context) StreamResponse(response string) error {
	b := strings.Builder{}
	b.WriteString("data: ")
	b.WriteString(response)
	b.WriteString("\n\n")
	_, err := c.ResponseWriter.Write([]byte(b.String()))
	if err != nil {
		return err
	}
	return nil
}

// BodyJson get json body from request and return map
// USAGE : data := c.BodyJson(r)
func (c *Context) BodyJson() map[string]any {
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

func (c *Context) BodyText() string {
	defer c.Request.Body.Close()
	b, err := io.ReadAll(c.Request.Body)
	if logger.CheckError(err) {
		return ""
	}
	return string(b)
}

// Redirect redirect the client to the specified path with a custom code
func (c *Context) Redirect(path string) {
	if c.status == 0 {
		c.status = http.StatusTemporaryRedirect
	}
	http.Redirect(c.ResponseWriter, c.Request, path, c.status)
}

// ServeFile serve a file from handler
func (c *Context) ServeFile(content_type, path_to_file string) {
	c.SetHeader("Content-Type", content_type)
	http.ServeFile(c.ResponseWriter, c.Request, path_to_file)
}

// ServeEmbededFile serve an embeded file from handler
func (c *Context) ServeEmbededFile(content_type string, embed_file []byte) {
	c.SetHeader("Content-Type", content_type)
	_, _ = c.ResponseWriter.Write(embed_file)
}

// UploadFile upload received_filename into folder_out and return url,fileByte,error
func (c *Context) UploadFile(received_filename, folder_out string, acceptedFormats ...string) (string, []byte, error) {
	c.Request.ParseMultipartForm(int64(MultipartSize)) //10Mb
	defer func ()  {
		err := c.Request.MultipartForm.RemoveAll()
		logger.CheckError(err)
	}()
	var buff bytes.Buffer
	file, header, err := c.Request.FormFile(received_filename)
	if logger.CheckError(err) {
		return "", nil, err
	}
	defer file.Close()
	// copy the uploaded file to the buffer
	if _, err := io.Copy(&buff, file); err != nil {
		return "", nil, err
	}

	data_string := buff.String()

	// make DIRS if not exist
	err = os.MkdirAll(settings.MEDIA_DIR+"/"+folder_out+"/", 0664)
	if err != nil {
		return "", nil, err
	}
	// make file
	if len(acceptedFormats) == 0 {
		acceptedFormats = []string{"jpg", "jpeg", "png", "json"}
	}
	if utils.StringContains(header.Filename, acceptedFormats...) {
		dst, err := os.Create(settings.MEDIA_DIR+"/" + folder_out + "/" + header.Filename)
		if err != nil {
			return "", nil, err
		}
		defer dst.Close()
		dst.Write([]byte(data_string))

		url := settings.MEDIA_DIR+"/" + folder_out + "/" + header.Filename
		return url, []byte(data_string), nil
	} else {
		return "", nil, fmt.Errorf("expecting filename to finish to be %v", acceptedFormats)
	}
}

func (c *Context) UploadFiles(received_filenames []string, folder_out string, acceptedFormats ...string) ([]string, [][]byte, error) {
	_, formFiles := utils.ParseMultipartForm(c.Request)
	urls := []string{}
	datas := [][]byte{}
	for inputName, files := range formFiles {
		var buff bytes.Buffer
		if len(files) > 0 && utils.SliceContains(received_filenames, inputName) {
			for _, f := range files {
				file, err := f.Open()
				if logger.CheckError(err) {
					return nil, nil, err
				}
				defer file.Close()
				// copy the uploaded file to the buffer
				if _, err := io.Copy(&buff, file); err != nil {
					return nil, nil, err
				}

				data_string := buff.String()

				// make DIRS if not exist
				err = os.MkdirAll(settings.MEDIA_DIR+"/"+folder_out+"/", 0664)
				if err != nil {
					return nil, nil, err
				}
				// make file
				if len(acceptedFormats) == 0 {
					acceptedFormats = []string{"jpg", "jpeg", "png", "json"}
				}
				if utils.StringContains(f.Filename, acceptedFormats...) {
					dst, err := os.Create(settings.MEDIA_DIR+"/" + folder_out + "/" + f.Filename)
					if err != nil {
						return nil, nil, err
					}
					defer dst.Close()
					dst.Write([]byte(data_string))

					url := settings.MEDIA_DIR+"/" + folder_out + "/" + f.Filename
					urls = append(urls, url)
					datas = append(datas, []byte(data_string))
				} else {
					logger.Info(f.Filename, "not handled")
					return nil, nil, fmt.Errorf("expecting filename to finish to be %v", acceptedFormats)
				}
			}
		}

	}
	return urls, datas, nil
}

// DELETE FILE
func (c *Context) DeleteFile(path string) error {
	err := os.Remove("." + path)
	if err != nil {
		return err
	} else {
		return nil
	}
}

// Download download data_bytes(content) asFilename(test.json,data.csv,...) to the client
func (c *Context) Download(data_bytes []byte, asFilename string) {
	bytesReader := bytes.NewReader(data_bytes)
	c.SetHeader("Content-Disposition", "attachment; filename="+strconv.Quote(asFilename))
	c.SetHeader("Content-Type", c.Request.Header.Get("Content-Type"))
	io.Copy(c.ResponseWriter, bytesReader)
}

// EnableTranslations get user ip, then location country using nmap, so don't use it if u don't have it install, and then it parse csv file to find the language spoken in this country, to finaly set cookie 'lang' to 'en' or 'fr'...
func (c *Context) EnableTranslations() {
	ip := c.GetUserIP()
	if utils.StringContains(ip, "127.0.0.1", "localhost", "") {
		c.SetCookie("lang", "en")
		return
	}
	country := utils.GetIpCountry(ip)
	if country != "" {
		if v, ok := mCountryLanguage.Get(country); ok {
			c.SetCookie("lang", v)
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

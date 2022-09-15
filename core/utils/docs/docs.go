package docs

import (
	"encoding/json"
	"io/fs"
	"os"

	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils/logger"
)

var path = settings.STATIC_DIR + "/docs/docs.json"

func New() *Docs {
	// read the file
	bytee, err := os.ReadFile(path)
	if logger.CheckError(err) {
		return &Docs{
			OpenApi: "3.0.1",
			Info: Info{
				Title:       "KaGo Docs",
				Description: "KaGo docs, ready to use with internal 'docs' library",
				Contact: Contact{
					Email: "kamalshkeir@gmail.com",
					Url:   "https://kamalshkeir.github.io",
				},
				Version: "1.0.0",
			},
			Host: "localhost:9313",
			ExternalDocs: ExternalDocs{
				Description: "Send me email on kamalshkeir@gmail.com",
				Url:         "https://kamalshkeir.github.io",
			},
			Servers: []Server{
				{Url: "http://localhost:9313"},
			},
			Tags:       []Tag{},
			Paths:      map[string]map[string]Path{},
			Components: map[string]map[string]Model{},
		}
	}
	docss := &Docs{
		OpenApi: "3.0.1",
		Info: Info{
			Title:       "KaGo Docs",
			Description: "KaGo docs, ready to use with internal 'docs' library",
			Contact: Contact{
				Email: "kamalshkeir@gmail.com",
				Url:   "https://kamalshkeir.github.io",
			},
			Version: "1.0.0",
		},
		Host: "localhost:9313",
		ExternalDocs: ExternalDocs{
			Description: "Send me email on kamalshkeir@gmail.com",
			Url:         "https://kamalshkeir.github.io",
		},
		Servers: []Server{
			{Url: "http://localhost:9313"},
		},
		Tags:       []Tag{},
		Paths:      map[string]map[string]Path{},
		Components: map[string]map[string]Model{},
	}
	// load it into data
	err = json.Unmarshal(bytee, docss)
	logger.CheckError(err)
	return docss
}

func (docs *Docs) String() string {
	// read the file
	docs.m.RLock()
	defer docs.m.RUnlock()
	b, err := json.MarshalIndent(docs, "", "  ")
	if logger.CheckError(err) {
		return ""
	}
	return string(b)
}

func (docs *Docs) Save() {
	docs.m.RLock()
	byte_again, _ := json.MarshalIndent(docs, "", "\t")
	docs.m.RUnlock()
	err := os.WriteFile(path, byte_again, fs.ModePerm)
	logger.CheckError(err)
}

func (docs *Docs) AddPath(path, method string, p Path) {
	docs.m.Lock()
	if _, ok := docs.Paths[path]; !ok {
		docs.Paths[path] = map[string]Path{}
	}
	docs.Paths[path][method] = p
	docs.m.Unlock()
}

func (docs *Docs) RemovePath(path string, method ...string) {
	docs.m.Lock()
	if len(method) == 0 {
		delete(docs.Paths, path)
	} else {
		delete(docs.Paths[path], method[0])
		if len(docs.Paths[path]) == 0 {
			delete(docs.Paths, path)
		}
	}
	docs.m.Unlock()
}

func (docs *Docs) AddTag(t Tag) {
	docs.m.Lock()
	if len(docs.Tags) == 0 {
		docs.Tags = []Tag{t}
		docs.m.Unlock()
		return
	}
	for _, tag := range docs.Tags {
		if tag.Name == t.Name {
			docs.m.Unlock()
			return
		}
	}
	docs.Tags = append(docs.Tags, t)
	docs.m.Unlock()
}

func (docs *Docs) RemoveTag(tagName string) {
	docs.m.Lock()
	for i, tag := range docs.Tags {
		if tag.Name == tagName {
			docs.Tags = append(docs.Tags[:i], docs.Tags[i+1:]...)
		}
	}
	docs.m.Unlock()
}

func (docs *Docs) AddModel(name string, m Model) {
	docs.m.Lock()
	if _, ok := docs.Components["schemas"]; !ok {
		docs.Components["schemas"] = map[string]Model{
			name: m,
		}
		docs.m.Unlock()
		return
	}
	docs.Components["schemas"][name] = m
	docs.m.Unlock()
}

func (docs *Docs) RemoveModel(modelName string) {
	docs.m.Lock()
	if sch, ok := docs.Components["schemas"]; ok {
		if _, ok := sch[modelName]; ok {
			delete(docs.Components[modelName], modelName)
		}
	}
	docs.m.Unlock()
}

/* doc := docs.New()

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
doc.Save() */

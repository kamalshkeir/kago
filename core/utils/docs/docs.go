package docs

import (
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"strings"

	"github.com/kamalshkeir/kago/core/utils"
	"github.com/kamalshkeir/kago/core/utils/logger"
)

const path = "assets/static/docs/docs.json"

func New() *Docs {
	return &Docs{}
}

func (docs *Docs) Read() *Docs {	
	// read the file
	byte,err := ioutil.ReadFile(path)
	if logger.CheckError(err) {
		return &Docs{
			BasePath: "/",
			SwaggerVersion: "2.0",
			Host: "localhost:9313",
			ExternalDocs: ExternalDocs{
				Description: "Send me email on kamalshkeir@gmail.com",
				Url: "https://kamalshkeir.github.io",
			},
			Info: Info{
				Contact: Contact{
					Email: "kamalshkeir@gmail.com",
					Url:"https://kamalshkeir.github.io",
				},
				Description: "KamFram docs, ready to use with internal 'docs' library",
				Title: "KamFram Docs",
				Version: "1.0.0",
			},
			Schemes: []string{"http"},
		}
	}
	docss := &Docs{
		BasePath: "/",
		SwaggerVersion: "2.0",
		Host: "localhost:9313",
		ExternalDocs: ExternalDocs{
			Description: "Send me email on kamalshkeir@gmail.com",
			Url: "https://kamalshkeir.github.io",
		},
		Info: Info{
			Contact: Contact{
				Email: "kamalshkeir@gmail.com",
				Url:"https://kamalshkeir.github.io",
			},
			Description: "KamFram docs, ready to use with internal 'docs' library",
			Title: "KamFram Docs",
			Version: "1.0.0",
		},
		Schemes: []string{"http"},
	}
	// load it into data
	err = json.Unmarshal(byte,docss)
	logger.CheckError(err)
	return docss
}

func (docs *Docs) Save() {	
	byte_again,_ := json.MarshalIndent(docs,"","\t")
	err := ioutil.WriteFile(path,byte_again,fs.ModePerm)
	logger.CheckError(err)
}

func NewQueryParam(paramName,paramType,paramDesc string,required bool) Param {
	return Param{
		In: "query",
		Name: paramName,
		Schema: Schema{
			Type: paramType,
		},
		Required: required,
		Description: paramDesc,
	}
}

func NewPathParam(paramName,paramType,paramDesc string,required bool) Param {
	return Param{
		In: "path",
		Name: paramName,
		Schema: Schema{
			Type: paramType,
		},
		Required: required,
		Description: paramDesc,
	}
}

func NewBodyParam(paramName,refName,paramDesc string,required bool) Param {
	return Param{
		In: "body",
		Name: paramName,
		Schema: Schema{
			Type: "object",
			Ref: "#/definitions/"+refName,
		},
		Required: required,
		Description: paramDesc,
	}
}

func (docs *Docs) SetVersion(version string){
	docs.Info.Version=version
}

func (docs *Docs) AddTag(name string, desc string) {
	for i,tag := range docs.Tags {
		if tag.Name == name {
			docs.Tags = append(docs.Tags[:i],docs.Tags[i+1:]...)
		}
	}
	docs.Tags = append(docs.Tags, Tag{
		Name: name,
		Description: desc,
	})
}

func (docs *Docs) Print() {
	logger.Info("Version:",docs.Info.Version)
	logger.Success("--------------------------------------")
	logger.Info("Tags:",docs.Tags)
	logger.Success("--------------------------------------")
	logger.Info("Models:")
	for name,def := range docs.Definitions {
		logger.Info(name,"-->",def)
	}
	logger.Success("--------------------------------------")
	logger.Info("Paths:")
	for url,maap := range docs.Paths {
		for method,path := range maap {
			logger.Info(method,url,path)
		}
	}
}

func (docs *Docs) RemoveTag(name string) {
	for i,tag := range docs.Tags {
		if tag.Name == name {
			docs.Tags = append(docs.Tags[:i],docs.Tags[i+1:]...)
		}
	}
}

func (docs *Docs) AddModel(model_name string, example_map map[string]any, fields_props map[string]Property) {
	if model,ok := docs.Definitions[model_name];ok {
		model.Example = example_map
		model.Properties = fields_props
		model.Type = "object"
	} else {
		docs.Definitions[model_name]=Definition{
			Example: example_map,
			Properties: fields_props,
			Type: "object",
		}
	}
}

func (docs *Docs) RemoveModel(model_name string) {
	if _,ok := docs.Definitions[model_name];ok {
		delete(docs.Definitions,model_name)
	}
}

func (docs *Docs) AddPath(
	url,method,desc,tag string,
	params []Param, 
	tags []string, 
	ref_model string,
	) {
	if p,ok := docs.Paths[url];ok {
		// path exist
		if meth,ok := p[method];ok {
			// method exist
			meth.Description=desc
			meth.Summary=desc
			// if param exist delete it
			meth.Parameters = append([]Param{}, params...)
			// if tag exist delete it
			meth.Tags=append([]string{}, tags...)
			if !strings.EqualFold(method,"get") {
				meth.Consumes=[]string{"application/json"}
				meth.Produces=[]string{"application/json"}
				meth.OperationId=utils.GenerateRandomString(6)
			}
			meth.Responses=map[string]Response{
				"200":{
					Description: "OK",
				},
				"404":{
					Description: "NOT FOUND",
				},
			}
		} else {
			// method doesn't exist
			// path doesn't exist
			var cons []string
			var opId string
			if !strings.EqualFold(method,"get") {
				cons=[]string{"application/json"}
				opId=utils.GenerateRandomString(6)
			}
			p[method]=Path{
				Description: desc,
				Summary: desc,
				OperationId: opId,
				Consumes: cons,
				Produces: cons,
				Parameters: append([]Param{}, params...),
				Tags: append([]string{}, tags...),
				Responses: map[string]Response{
					"200":{
						Description: "OK",
					},
					"404":{
						Description: "NOT FOUND",
					},
				},
			}
		}
	} else {
		// path doesn't exist
		var cons []string
		var opId string
		if !strings.EqualFold(method,"get") {
			cons=[]string{"application/json"}
			opId=utils.GenerateRandomString(6)
		}
		docs.Paths[url]=map[string]Path{}
		docs.Paths[url][method]=Path{
			Description: desc,
			Summary: desc,
			OperationId: opId,
			Consumes: cons,
			Produces: cons,
			Tags: append([]string{},tags...),
			Parameters: append([]Param{},params...),
			Responses: map[string]Response{
				"200":{
					Description: "OK",
				},
				"404":{
					Description: "NOT FOUND",
				},
			},
		}
		
	}

}

func (docs *Docs) RemoveMethodFromPath(path string, method string) {
	if p,ok := docs.Paths[path];ok {
		if _,ok := p[method];ok {
			delete(docs.Paths[path],method)
		}
	}
}

func (docs *Docs) RemovePath(path string) {
	if _,ok := docs.Paths[path];ok {
		delete(docs.Paths,path)
	}
}

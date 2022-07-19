package docs

import "sync"

type Docs struct {
	OpenApi      string                      `json:"openapi,omitempty"`
	Host         string                      `json:"host,omitempty"`
	Info         Info                        `json:"info,omitempty"`
	ExternalDocs ExternalDocs                `json:"externalDocs,omitempty"`
	Servers      []Server                    `json:"servers,omitempty"`
	Tags         []Tag                       `json:"tags,omitempty"`
	Paths        map[string]map[string]Path  `json:"paths,omitempty"`
	Components   map[string]map[string]Model `json:"components,omitempty"`
	m            sync.RWMutex
}

type Property struct {
	Description string     `json:"description,omitempty"`
	Required    bool       `json:"required,omitempty"`
	Type        string     `json:"type"`
	Format      string     `json:"format,omitempty"`
	Enum        []string   `json:"enum,omitempty"`
	Default     any        `json:"default,omitempty"`
	Example     any        `json:"example,omitempty"`
	Ref         string     `json:"$ref,omitempty"`
	ArrayItem   ArrayItems `json:"items,omitempty"`
}

type ArrayItems struct {
	Type    string `json:"type,omitempty"`
	Ref     string `json:"$ref,omitempty"`
	Default any    `json:"default,omitempty"`
}

type ExternalDocs struct {
	Description string `json:"description,omitempty"`
	Url         string `json:"url,omitempty"`
}

type Contact struct {
	Email string `json:"email,omitempty"`
	Url   string `json:"url,omitempty"`
}

type Info struct {
	Contact     Contact `json:"contact,omitempty"`
	Description string  `json:"description,omitempty"`
	Title       string  `json:"title,omitempty"`
	Version     string  `json:"version,omitempty"`
}

type Response struct {
	Description string `json:"description,omitempty"`
}

type Model struct {
	Type           string              `json:"type,omitempty"`
	Properties     map[string]Property `json:"properties,omitempty"`
	Xml            XmL                 `json:"xml,omitempty"`
	RequiredFields []string            `json:"required,omitempty"`
}

type XmL struct {
	Name string `json:"name,omitempty"`
}

type Param struct {
	Name        string `json:"name,omitempty"`
	In          string `json:"in,omitempty"`
	Description string `json:"description,omitempty"`
	Schema      Schema `json:"schema,omitempty"`
	Style       string `json:"style,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Explode     bool   `json:"explode,omitempty"`
}

type Path struct {
	Tags        []string            `json:"tags,omitempty"`
	Summary     string              `json:"summary,omitempty"`
	OperationId string              `json:"operationId,omitempty"`
	Requestbody RequestBody         `json:"requestBody,omitempty"`
	Description string              `json:"description,omitempty"`
	Parameters  []Param             `json:"parameters,omitempty"`
	Responses   map[string]Response `json:"responses,omitempty"`
	Consumes    []string            `json:"consumes,omitempty"`
	Produces    []string            `json:"produces,omitempty"`
}

type RequestBody struct {
	Description string                 `json:"description,omitempty"`
	Content     map[string]ContentType `json:"content,omitempty"`
	Required    bool                   `json:"required,omitempty"`
}

type ContentType struct {
	Schema Schema `json:"schema,omitempty"`
}

type Schema struct {
	Type       string     `json:"type,omitempty"`
	Format     string     `json:"format,omitempty"`
	Ref        string     `json:"$ref,omitempty"`
	ArrayItems ArrayItems `json:"items,omitempty"`
}

type Tag struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type Server struct {
	Url string `json:"url,omitempty"`
}

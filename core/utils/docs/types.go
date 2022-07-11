package docs

type Property struct {
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Type        string `json:"type"`
	Format      string `json:"format,omitempty"`
}

type ExternalDocs struct {
	Description string `json:"description"`
	Url         string `json:"url"`
}

type Contact struct {
	Email string `json:"email"`
	Url   string `json:"url"`
}
type Info struct {
	Contact     Contact `json:"contact"`
	Description string  `json:"description"`
	Title       string  `json:"title"`
	Version     string  `json:"version"`
}

type Response struct {
	Description string `json:"description"`
}

type Schema struct {
	Ref  string `json:"$ref,omitempty"`
	Type string `json:"type,omitempty"`
}

type Param struct {
	Description string `json:"description"`
	In          string `json:"in"`
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Schema      Schema `json:"schema"`
}

type Path struct {
	Description string              `json:"description"`
	Parameters  []Param             `json:"parameters"`
	Responses   map[string]Response `json:"responses"`
	Summary     string              `json:"summary"`
	Tags        []string            `json:"tags"`
	Consumes    []string            `json:"consumes,omitempty"`
	Produces    []string            `json:"produces,omitempty"`
	OperationId string              `json:"operationId,omitempty"`
}

type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Definition struct {
	Example    map[string]any      `json:"example"`
	Properties map[string]Property `json:"properties"`
	Type       string              `json:"type"`
}

type Docs struct {
	SwaggerVersion string                     `json:"swagger"`
	BasePath       string                     `json:"basePath"`
	Definitions    map[string]Definition      `json:"definitions"`
	ExternalDocs   ExternalDocs               `json:"externalDocs"`
	Host           string                     `json:"host"`
	Info           Info                       `json:"info"`
	Paths          map[string]map[string]Path `json:"paths"`
	Schemes        []string                   `json:"schemes"`
	Tags           []Tag                      `json:"tags"`
}
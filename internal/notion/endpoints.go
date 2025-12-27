// internal/notion/endpoints.go
package notion

// Endpoint represents an API endpoint with metadata.
type Endpoint struct {
	Method     string
	Path       string
	Idempotent bool // Safe to retry on 5xx
	RateLimit  int  // Requests per second (0 = unknown)
}

// EndpointRegistry holds all API endpoints.
type EndpointRegistry struct {
	Pages       PagesEndpoints
	Blocks      BlocksEndpoints
	Databases   DatabasesEndpoints
	DataSources DataSourcesEndpoints
	Users       UsersEndpoints
	Search      Endpoint
	Comments    CommentsEndpoints
}

type PagesEndpoints struct {
	Get    Endpoint
	Create Endpoint
	Update Endpoint
}

type BlocksEndpoints struct {
	Get      Endpoint
	Update   Endpoint
	Delete   Endpoint
	Children Endpoint
	Append   Endpoint
}

type DatabasesEndpoints struct {
	Get    Endpoint
	Create Endpoint
	Update Endpoint
	Query  Endpoint
}

type DataSourcesEndpoints struct {
	Get       Endpoint
	Create    Endpoint
	Update    Endpoint
	Query     Endpoint
	Templates Endpoint
}

type UsersEndpoints struct {
	Get  Endpoint
	List Endpoint
	Me   Endpoint
}

type CommentsEndpoints struct {
	Get    Endpoint
	Create Endpoint
	List   Endpoint
}

// Endpoints is the global registry of all API endpoints.
var Endpoints = EndpointRegistry{
	Pages: PagesEndpoints{
		Get:    Endpoint{Method: "GET", Path: "/pages/{id}", Idempotent: true},
		Create: Endpoint{Method: "POST", Path: "/pages", Idempotent: false},
		Update: Endpoint{Method: "PATCH", Path: "/pages/{id}", Idempotent: false},
	},
	Blocks: BlocksEndpoints{
		Get:      Endpoint{Method: "GET", Path: "/blocks/{id}", Idempotent: true},
		Update:   Endpoint{Method: "PATCH", Path: "/blocks/{id}", Idempotent: false},
		Delete:   Endpoint{Method: "DELETE", Path: "/blocks/{id}", Idempotent: false},
		Children: Endpoint{Method: "GET", Path: "/blocks/{id}/children", Idempotent: true},
		Append:   Endpoint{Method: "PATCH", Path: "/blocks/{id}/children", Idempotent: false},
	},
	Databases: DatabasesEndpoints{
		Get:    Endpoint{Method: "GET", Path: "/databases/{id}", Idempotent: true},
		Create: Endpoint{Method: "POST", Path: "/databases", Idempotent: false},
		Update: Endpoint{Method: "PATCH", Path: "/databases/{id}", Idempotent: false},
		Query:  Endpoint{Method: "POST", Path: "/databases/{id}/query", Idempotent: true},
	},
	DataSources: DataSourcesEndpoints{
		Get:       Endpoint{Method: "GET", Path: "/data_sources/{id}", Idempotent: true},
		Create:    Endpoint{Method: "POST", Path: "/data_sources", Idempotent: false},
		Update:    Endpoint{Method: "PATCH", Path: "/data_sources/{id}", Idempotent: false},
		Query:     Endpoint{Method: "POST", Path: "/data_sources/{id}/query", Idempotent: true},
		Templates: Endpoint{Method: "GET", Path: "/data_sources/templates", Idempotent: true},
	},
	Users: UsersEndpoints{
		Get:  Endpoint{Method: "GET", Path: "/users/{id}", Idempotent: true},
		List: Endpoint{Method: "GET", Path: "/users", Idempotent: true},
		Me:   Endpoint{Method: "GET", Path: "/users/me", Idempotent: true},
	},
	Search: Endpoint{Method: "POST", Path: "/search", Idempotent: true},
	Comments: CommentsEndpoints{
		Get:    Endpoint{Method: "GET", Path: "/comments/{id}", Idempotent: true},
		Create: Endpoint{Method: "POST", Path: "/comments", Idempotent: false},
		List:   Endpoint{Method: "GET", Path: "/comments", Idempotent: true},
	},
}

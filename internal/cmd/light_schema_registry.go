package cmd

import "sort"

type lightSchema struct {
	Command    string
	ItemFields []string
	Guarantees []string
}

var (
	lightFieldsEntityRef = []string{"id", "object", "title", "url"}
	lightFieldsPage      = []string{"id", "object", "title", "url"}
	lightFieldsUser      = []string{"id", "name", "email", "type"}
	lightFieldsComment   = []string{"id", "discussion_id", "parent_id", "created_by_id", "created_by", "created_time", "text"}
	lightFieldsFile      = []string{"id", "file_name", "status", "size", "created_time", "expiry_time"}
)

// lightSchemaRegistry defines the compact payload contract for commands that expose --light/--li.
// ItemFields are the only guaranteed fields for each light object entry.
var lightSchemaRegistry = map[string]lightSchema{
	"list": {
		Command:    "list",
		ItemFields: lightFieldsEntityRef,
		Guarantees: []string{
			"Top-level alias for page list",
			"Returns same light item shape as page list/search",
		},
	},
	"search": {
		Command:    "search",
		ItemFields: lightFieldsEntityRef,
		Guarantees: []string{
			"Normalizes object=data_source to object=ds",
			"Extracts plain-text title from title/properties payloads",
		},
	},
	"page list": {
		Command:    "page list",
		ItemFields: lightFieldsEntityRef,
		Guarantees: []string{
			"Delegates to search with filter=page",
			"Returns same light item shape as search",
		},
	},
	"db list": {
		Command:    "db list",
		ItemFields: lightFieldsEntityRef,
		Guarantees: []string{
			"Returns data source search results in compact form",
			"Normalizes object=data_source to object=ds",
		},
	},
	"datasource list": {
		Command:    "datasource list",
		ItemFields: lightFieldsEntityRef,
		Guarantees: []string{
			"Returns data source search results in compact form",
			"Normalizes object=data_source to object=ds",
		},
	},
	"page get": {
		Command:    "page get",
		ItemFields: lightFieldsPage,
		Guarantees: []string{
			"Defaults object to page when page object is missing",
			"Omits full properties payload",
		},
	},
	"user list": {
		Command:    "user list",
		ItemFields: lightFieldsUser,
		Guarantees: []string{
			"Includes person email when present",
			"Omit avatar/person/bot details",
		},
	},
	"comment list": {
		Command:    "comment list",
		ItemFields: lightFieldsComment,
		Guarantees: []string{
			"Flattens rich_text into plain-text text",
			"Omit full rich_text array",
		},
	},
	"file list": {
		Command:    "file list",
		ItemFields: lightFieldsFile,
		Guarantees: []string{
			"Omit upload URL and transport metadata",
		},
	},
}

func lightSchemaCommands() []string {
	commands := make([]string, 0, len(lightSchemaRegistry))
	for command := range lightSchemaRegistry {
		commands = append(commands, command)
	}
	sort.Strings(commands)
	return commands
}

func lookupLightSchema(command string) (lightSchema, bool) {
	schema, ok := lightSchemaRegistry[command]
	return schema, ok
}

package notion

// FormulaProperty represents a formula property value.
// Formula values are read-only and computed by Notion.
type FormulaProperty struct {
	Type    string       `json:"type"`
	Formula FormulaValue `json:"formula"`
}

// FormulaValue contains the computed formula result.
type FormulaValue struct {
	Type    string   `json:"type"` // "string", "number", "boolean", "date"
	String  *string  `json:"string,omitempty"`
	Number  *float64 `json:"number,omitempty"`
	Boolean *bool    `json:"boolean,omitempty"`
	Date    *Date    `json:"date,omitempty"`
}

// Date represents a date value in Notion.
type Date struct {
	Start    string  `json:"start"`
	End      *string `json:"end,omitempty"`
	TimeZone *string `json:"time_zone,omitempty"`
}

// RollupProperty represents a rollup property value.
// Rollup values are read-only and computed by aggregating related entries.
type RollupProperty struct {
	Type   string      `json:"type"`
	Rollup RollupValue `json:"rollup"`
}

// RollupValue contains the computed rollup result.
type RollupValue struct {
	Type     string                   `json:"type"` // "array", "date", "incomplete", "number", "unsupported"
	Function string                   `json:"function"`
	Array    []map[string]interface{} `json:"array"`
	Number   *float64                 `json:"number,omitempty"`
	Date     *Date                    `json:"date,omitempty"`
}

// RollupFunction constants for rollup aggregation functions.
const (
	RollupFunctionCount            = "count"
	RollupFunctionCountValues      = "count_values"
	RollupFunctionEmpty            = "empty"
	RollupFunctionNotEmpty         = "not_empty"
	RollupFunctionPercentEmpty     = "percent_empty"
	RollupFunctionPercentNotEmpty  = "percent_not_empty"
	RollupFunctionSum              = "sum"
	RollupFunctionAverage          = "average"
	RollupFunctionMedian           = "median"
	RollupFunctionMin              = "min"
	RollupFunctionMax              = "max"
	RollupFunctionRange            = "range"
	RollupFunctionShowOriginal     = "show_original"
	RollupFunctionShowUnique       = "show_unique"
	RollupFunctionEarliest         = "earliest_date"
	RollupFunctionLatest           = "latest_date"
	RollupFunctionDateRange        = "date_range"
	RollupFunctionChecked          = "checked"
	RollupFunctionUnchecked        = "unchecked"
	RollupFunctionPercentChecked   = "percent_checked"
	RollupFunctionPercentUnchecked = "percent_unchecked"
)

// PropertyValue is a generic property value that can hold any property type.
type PropertyValue struct {
	ID      string        `json:"id,omitempty"`
	Type    string        `json:"type"`
	Formula *FormulaValue `json:"formula,omitempty"`
	Rollup  *RollupValue  `json:"rollup,omitempty"`
	// Other common property types
	Title          []RichText               `json:"title"`
	RichText       []RichText               `json:"rich_text"`
	Number         *float64                 `json:"number,omitempty"`
	Select         *SelectOption            `json:"select,omitempty"`
	MultiSelect    []SelectOption           `json:"multi_select"`
	Date           *Date                    `json:"date,omitempty"`
	Checkbox       *bool                    `json:"checkbox,omitempty"`
	URL            *string                  `json:"url,omitempty"`
	Email          *string                  `json:"email,omitempty"`
	PhoneNumber    *string                  `json:"phone_number,omitempty"`
	Files          []FileReference          `json:"files"`
	Relation       []RelationReference      `json:"relation"`
	People         []map[string]interface{} `json:"people"`
	CreatedTime    *string                  `json:"created_time,omitempty"`
	LastEditedTime *string                  `json:"last_edited_time,omitempty"`
	CreatedBy      map[string]interface{}   `json:"created_by,omitempty"`
	LastEditedBy   map[string]interface{}   `json:"last_edited_by,omitempty"`
	Status         *StatusOption            `json:"status,omitempty"`
	UniqueID       *UniqueID                `json:"unique_id,omitempty"`
}

// SelectOption represents a select or multi-select option.
type SelectOption struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// StatusOption represents a status property option.
type StatusOption struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Color       string `json:"color,omitempty"`
	Description string `json:"description,omitempty"`
}

// FileReference represents a file in a files property.
type FileReference struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"` // "file" or "external"
	File     map[string]interface{} `json:"file,omitempty"`
	External map[string]interface{} `json:"external,omitempty"`
}

// RelationReference represents a relation to another page.
type RelationReference struct {
	ID string `json:"id"`
}

// UniqueID represents an auto-generated unique ID property.
type UniqueID struct {
	Prefix *string `json:"prefix,omitempty"`
	Number int     `json:"number"`
}

// GetFormulaString returns the string value of a formula, or empty string if not a string formula.
func (f *FormulaValue) GetFormulaString() string {
	if f == nil {
		return ""
	}
	if f.Type == "string" && f.String != nil {
		return *f.String
	}
	return ""
}

// GetFormulaNumber returns the number value of a formula, or 0 if not a number formula.
func (f *FormulaValue) GetFormulaNumber() float64 {
	if f == nil {
		return 0
	}
	if f.Type == "number" && f.Number != nil {
		return *f.Number
	}
	return 0
}

// GetFormulaBool returns the boolean value of a formula, or false if not a boolean formula.
func (f *FormulaValue) GetFormulaBool() bool {
	if f == nil {
		return false
	}
	if f.Type == "boolean" && f.Boolean != nil {
		return *f.Boolean
	}
	return false
}

// GetRollupNumber returns the number value of a rollup, or 0 if not a number rollup.
func (r *RollupValue) GetRollupNumber() float64 {
	if r == nil {
		return 0
	}
	if r.Type == "number" && r.Number != nil {
		return *r.Number
	}
	return 0
}

// GetRollupArrayLength returns the length of a rollup array, or 0 if not an array rollup.
func (r *RollupValue) GetRollupArrayLength() int {
	if r == nil {
		return 0
	}
	if r.Type == "array" {
		return len(r.Array)
	}
	return 0
}

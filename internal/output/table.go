package output

// Table represents a pre-rendered table for table output formatting.
type Table struct {
	Headers []string   `json:"headers,omitempty" yaml:"headers,omitempty"`
	Rows    [][]string `json:"rows,omitempty" yaml:"rows,omitempty"`
}

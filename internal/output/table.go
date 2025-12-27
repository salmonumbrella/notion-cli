package output

// Table represents a pre-rendered table for table output formatting.
type Table struct {
	Headers []string   `json:"headers" yaml:"headers"`
	Rows    [][]string `json:"rows" yaml:"rows"`
}

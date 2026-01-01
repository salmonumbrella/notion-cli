package notion

import (
	"encoding/json"
	"testing"
)

func TestParseFormulaProperty_Number(t *testing.T) {
	data := []byte(`{
		"type": "formula",
		"formula": {
			"type": "number",
			"number": 42
		}
	}`)

	var prop FormulaProperty
	if err := json.Unmarshal(data, &prop); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if prop.Formula.Type != "number" {
		t.Errorf("expected type 'number', got %q", prop.Formula.Type)
	}
	if prop.Formula.Number == nil || *prop.Formula.Number != 42 {
		t.Errorf("expected number 42, got %v", prop.Formula.Number)
	}
}

func TestParseFormulaProperty_String(t *testing.T) {
	data := []byte(`{
		"type": "formula",
		"formula": {
			"type": "string",
			"string": "computed value"
		}
	}`)

	var prop FormulaProperty
	if err := json.Unmarshal(data, &prop); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if prop.Formula.Type != "string" {
		t.Errorf("expected type 'string', got %q", prop.Formula.Type)
	}
	if prop.Formula.String == nil || *prop.Formula.String != "computed value" {
		t.Errorf("expected string 'computed value', got %v", prop.Formula.String)
	}
}

func TestParseFormulaProperty_Boolean(t *testing.T) {
	data := []byte(`{
		"type": "formula",
		"formula": {
			"type": "boolean",
			"boolean": true
		}
	}`)

	var prop FormulaProperty
	if err := json.Unmarshal(data, &prop); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if prop.Formula.Boolean == nil || *prop.Formula.Boolean != true {
		t.Errorf("expected boolean true, got %v", prop.Formula.Boolean)
	}
}

func TestParseRollupProperty_Number(t *testing.T) {
	data := []byte(`{
		"type": "rollup",
		"rollup": {
			"type": "number",
			"number": 100,
			"function": "sum"
		}
	}`)

	var prop RollupProperty
	if err := json.Unmarshal(data, &prop); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if prop.Rollup.Type != "number" {
		t.Errorf("expected type 'number', got %q", prop.Rollup.Type)
	}
	if prop.Rollup.Number == nil || *prop.Rollup.Number != 100 {
		t.Errorf("expected number 100, got %v", prop.Rollup.Number)
	}
	if prop.Rollup.Function != "sum" {
		t.Errorf("expected function 'sum', got %q", prop.Rollup.Function)
	}
}

func TestParseRollupProperty_Array(t *testing.T) {
	data := []byte(`{
		"type": "rollup",
		"rollup": {
			"type": "array",
			"array": [
				{"type": "title", "title": [{"text": {"content": "Item 1"}}]},
				{"type": "title", "title": [{"text": {"content": "Item 2"}}]}
			],
			"function": "show_original"
		}
	}`)

	var prop RollupProperty
	if err := json.Unmarshal(data, &prop); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if prop.Rollup.Type != "array" {
		t.Errorf("expected type 'array', got %q", prop.Rollup.Type)
	}
	if len(prop.Rollup.Array) != 2 {
		t.Errorf("expected 2 items, got %d", len(prop.Rollup.Array))
	}
}

func TestFormulaValue_GetString(t *testing.T) {
	str := "hello"
	f := &FormulaValue{Type: "string", String: &str}

	if f.GetFormulaString() != "hello" {
		t.Errorf("expected 'hello', got %q", f.GetFormulaString())
	}
}

func TestFormulaValue_GetNumber(t *testing.T) {
	num := 42.5
	f := &FormulaValue{Type: "number", Number: &num}

	if f.GetFormulaNumber() != 42.5 {
		t.Errorf("expected 42.5, got %f", f.GetFormulaNumber())
	}
}

func TestFormulaValue_GetBool(t *testing.T) {
	b := true
	f := &FormulaValue{Type: "boolean", Boolean: &b}

	if !f.GetFormulaBool() {
		t.Error("expected true")
	}
}

func TestRollupValue_GetNumber(t *testing.T) {
	num := 100.0
	r := &RollupValue{Type: "number", Number: &num, Function: "sum"}

	if r.GetRollupNumber() != 100.0 {
		t.Errorf("expected 100.0, got %f", r.GetRollupNumber())
	}
}

func TestRollupValue_GetArrayLength(t *testing.T) {
	r := &RollupValue{
		Type:     "array",
		Function: "show_original",
		Array: []map[string]interface{}{
			{"id": "1"},
			{"id": "2"},
			{"id": "3"},
		},
	}

	if r.GetRollupArrayLength() != 3 {
		t.Errorf("expected 3, got %d", r.GetRollupArrayLength())
	}
}

func TestPropertyValue_WithFormula(t *testing.T) {
	num := 42.0
	pv := PropertyValue{
		Type: "formula",
		Formula: &FormulaValue{
			Type:   "number",
			Number: &num,
		},
	}

	if pv.Formula == nil {
		t.Fatal("expected formula to be set")
	}
	if pv.Formula.GetFormulaNumber() != 42.0 {
		t.Errorf("expected 42.0, got %f", pv.Formula.GetFormulaNumber())
	}
}

func TestPropertyValue_WithRollup(t *testing.T) {
	num := 50.0
	pv := PropertyValue{
		Type: "rollup",
		Rollup: &RollupValue{
			Type:     "number",
			Number:   &num,
			Function: "average",
		},
	}

	if pv.Rollup == nil {
		t.Fatal("expected rollup to be set")
	}
	if pv.Rollup.Function != "average" {
		t.Errorf("expected function 'average', got %q", pv.Rollup.Function)
	}
}

func TestFormulaValue_GetString_WrongType(t *testing.T) {
	num := 42.0
	f := &FormulaValue{Type: "number", Number: &num}

	result := f.GetFormulaString()
	if result != "" {
		t.Errorf("expected empty string when calling GetFormulaString on number-type formula, got %q", result)
	}
}

func TestFormulaValue_GetNumber_WrongType(t *testing.T) {
	str := "hello"
	f := &FormulaValue{Type: "string", String: &str}

	result := f.GetFormulaNumber()
	if result != 0 {
		t.Errorf("expected 0 when calling GetFormulaNumber on string-type formula, got %f", result)
	}
}

func TestFormulaValue_GetBool_WrongType(t *testing.T) {
	str := "hello"
	f := &FormulaValue{Type: "string", String: &str}

	result := f.GetFormulaBool()
	if result != false {
		t.Error("expected false when calling GetFormulaBool on string-type formula")
	}
}

func TestRollupValue_GetNumber_WrongType(t *testing.T) {
	r := &RollupValue{
		Type:     "array",
		Function: "show_original",
		Array: []map[string]interface{}{
			{"id": "1"},
		},
	}

	result := r.GetRollupNumber()
	if result != 0 {
		t.Errorf("expected 0 when calling GetRollupNumber on array-type rollup, got %f", result)
	}
}

func TestFormulaValue_NilPointerFields(t *testing.T) {
	tests := []struct {
		name      string
		formula   *FormulaValue
		getString string
		getNumber float64
		getBool   bool
	}{
		{
			name:      "string type with nil String pointer",
			formula:   &FormulaValue{Type: "string", String: nil},
			getString: "",
			getNumber: 0,
			getBool:   false,
		},
		{
			name:      "number type with nil Number pointer",
			formula:   &FormulaValue{Type: "number", Number: nil},
			getString: "",
			getNumber: 0,
			getBool:   false,
		},
		{
			name:      "boolean type with nil Boolean pointer",
			formula:   &FormulaValue{Type: "boolean", Boolean: nil},
			getString: "",
			getNumber: 0,
			getBool:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := tt.formula.GetFormulaString(); result != tt.getString {
				t.Errorf("GetFormulaString() = %q, want %q", result, tt.getString)
			}
			if result := tt.formula.GetFormulaNumber(); result != tt.getNumber {
				t.Errorf("GetFormulaNumber() = %f, want %f", result, tt.getNumber)
			}
			if result := tt.formula.GetFormulaBool(); result != tt.getBool {
				t.Errorf("GetFormulaBool() = %v, want %v", result, tt.getBool)
			}
		})
	}
}

func TestRollupValue_EmptyArray(t *testing.T) {
	r := &RollupValue{
		Type:     "array",
		Function: "show_original",
		Array:    []map[string]interface{}{},
	}

	result := r.GetRollupArrayLength()
	if result != 0 {
		t.Errorf("expected 0 for empty array, got %d", result)
	}
}

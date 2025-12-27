package notion

import (
	"encoding/json"
	"testing"
)

func TestNewSyncedBlock_Original(t *testing.T) {
	children := []map[string]interface{}{
		{"type": "paragraph", "paragraph": map[string]interface{}{"rich_text": []interface{}{}}},
	}

	block := NewSyncedBlock(nil, children)

	if block["type"] != "synced_block" {
		t.Errorf("expected type 'synced_block', got %v", block["type"])
	}

	sb := block["synced_block"].(map[string]interface{})
	if sb["synced_from"] != nil {
		t.Error("expected synced_from to be nil for original block")
	}
	if sb["children"] == nil {
		t.Error("expected children to be set for original block")
	}
}

func TestNewSyncedBlock_Duplicate(t *testing.T) {
	sourceID := "source-block-123"

	block := NewSyncedBlock(&sourceID, nil)

	sb := block["synced_block"].(map[string]interface{})
	syncedFrom := sb["synced_from"].(map[string]interface{})

	if syncedFrom["type"] != "block_id" {
		t.Errorf("expected synced_from type 'block_id', got %v", syncedFrom["type"])
	}
	if syncedFrom["block_id"] != sourceID {
		t.Errorf("expected block_id %q, got %v", sourceID, syncedFrom["block_id"])
	}
}

func TestNewTableOfContents(t *testing.T) {
	block := NewTableOfContents("blue")

	if block["type"] != "table_of_contents" {
		t.Errorf("expected type 'table_of_contents', got %v", block["type"])
	}

	toc := block["table_of_contents"].(map[string]interface{})
	if toc["color"] != "blue" {
		t.Errorf("expected color 'blue', got %v", toc["color"])
	}
}

func TestNewBreadcrumb(t *testing.T) {
	block := NewBreadcrumb()

	if block["type"] != "breadcrumb" {
		t.Errorf("expected type 'breadcrumb', got %v", block["type"])
	}
}

func TestNewColumnList(t *testing.T) {
	col1Children := []map[string]interface{}{
		{"type": "paragraph", "paragraph": map[string]interface{}{}},
	}
	col2Children := []map[string]interface{}{
		{"type": "paragraph", "paragraph": map[string]interface{}{}},
	}

	block := NewColumnList(col1Children, col2Children)

	if block["type"] != "column_list" {
		t.Errorf("expected type 'column_list', got %v", block["type"])
	}

	children := block["column_list"].(map[string]interface{})["children"].([]map[string]interface{})
	if len(children) != 2 {
		t.Errorf("expected 2 columns, got %d", len(children))
	}
}

func TestNewLinkPreview(t *testing.T) {
	url := "https://example.com/page"
	block := NewLinkPreview(url)

	if block["type"] != "link_preview" {
		t.Errorf("expected type 'link_preview', got %v", block["type"])
	}

	lp := block["link_preview"].(map[string]interface{})
	if lp["url"] != url {
		t.Errorf("expected url %q, got %v", url, lp["url"])
	}
}

func TestNewParagraph(t *testing.T) {
	text := "Hello, world!"
	block := NewParagraph(text)

	if block["type"] != "paragraph" {
		t.Errorf("expected type 'paragraph', got %v", block["type"])
	}

	p := block["paragraph"].(map[string]interface{})
	rt := p["rich_text"].([]map[string]interface{})[0]
	content := rt["text"].(map[string]interface{})["content"]

	if content != text {
		t.Errorf("expected content %q, got %v", text, content)
	}
}

func TestNewCode(t *testing.T) {
	code := "func main() {}"
	lang := "go"
	block := NewCode(code, lang)

	if block["type"] != "code" {
		t.Errorf("expected type 'code', got %v", block["type"])
	}

	c := block["code"].(map[string]interface{})
	if c["language"] != lang {
		t.Errorf("expected language %q, got %v", lang, c["language"])
	}
}

func TestNewCallout(t *testing.T) {
	text := "Important note"
	emoji := "⚠️"
	block := NewCallout(text, emoji)

	if block["type"] != "callout" {
		t.Errorf("expected type 'callout', got %v", block["type"])
	}

	c := block["callout"].(map[string]interface{})
	icon := c["icon"].(map[string]interface{})
	if icon["emoji"] != emoji {
		t.Errorf("expected emoji %q, got %v", emoji, icon["emoji"])
	}
}

func TestNewHeading2(t *testing.T) {
	text := "Section Title"
	block := NewHeading2(text)

	if block["type"] != "heading_2" {
		t.Errorf("expected type 'heading_2', got %v", block["type"])
	}

	h2 := block["heading_2"].(map[string]interface{})
	rt := h2["rich_text"].([]map[string]interface{})[0]
	content := rt["text"].(map[string]interface{})["content"]

	if content != text {
		t.Errorf("expected content %q, got %v", text, content)
	}
}

func TestNewHeading3(t *testing.T) {
	text := "Subsection Title"
	block := NewHeading3(text)

	if block["type"] != "heading_3" {
		t.Errorf("expected type 'heading_3', got %v", block["type"])
	}

	h3 := block["heading_3"].(map[string]interface{})
	rt := h3["rich_text"].([]map[string]interface{})[0]
	content := rt["text"].(map[string]interface{})["content"]

	if content != text {
		t.Errorf("expected content %q, got %v", text, content)
	}
}

func TestNewToDo_Checked(t *testing.T) {
	text := "Complete task"
	block := NewToDo(text, true)

	if block["type"] != "to_do" {
		t.Errorf("expected type 'to_do', got %v", block["type"])
	}

	todo := block["to_do"].(map[string]interface{})
	rt := todo["rich_text"].([]map[string]interface{})[0]
	content := rt["text"].(map[string]interface{})["content"]

	if content != text {
		t.Errorf("expected content %q, got %v", text, content)
	}

	if todo["checked"] != true {
		t.Errorf("expected checked to be true, got %v", todo["checked"])
	}
}

func TestNewToDo_Unchecked(t *testing.T) {
	text := "Pending task"
	block := NewToDo(text, false)

	if block["type"] != "to_do" {
		t.Errorf("expected type 'to_do', got %v", block["type"])
	}

	todo := block["to_do"].(map[string]interface{})
	rt := todo["rich_text"].([]map[string]interface{})[0]
	content := rt["text"].(map[string]interface{})["content"]

	if content != text {
		t.Errorf("expected content %q, got %v", text, content)
	}

	if todo["checked"] != false {
		t.Errorf("expected checked to be false, got %v", todo["checked"])
	}
}

func TestNewQuote(t *testing.T) {
	text := "To be or not to be"
	block := NewQuote(text)

	if block["type"] != "quote" {
		t.Errorf("expected type 'quote', got %v", block["type"])
	}

	quote := block["quote"].(map[string]interface{})
	rt := quote["rich_text"].([]map[string]interface{})[0]
	content := rt["text"].(map[string]interface{})["content"]

	if content != text {
		t.Errorf("expected content %q, got %v", text, content)
	}
}

func TestNewBulletedListItem(t *testing.T) {
	text := "First item"
	block := NewBulletedListItem(text)

	if block["type"] != "bulleted_list_item" {
		t.Errorf("expected type 'bulleted_list_item', got %v", block["type"])
	}

	item := block["bulleted_list_item"].(map[string]interface{})
	rt := item["rich_text"].([]map[string]interface{})[0]
	content := rt["text"].(map[string]interface{})["content"]

	if content != text {
		t.Errorf("expected content %q, got %v", text, content)
	}
}

func TestNewNumberedListItem(t *testing.T) {
	text := "First step"
	block := NewNumberedListItem(text)

	if block["type"] != "numbered_list_item" {
		t.Errorf("expected type 'numbered_list_item', got %v", block["type"])
	}

	item := block["numbered_list_item"].(map[string]interface{})
	rt := item["rich_text"].([]map[string]interface{})[0]
	content := rt["text"].(map[string]interface{})["content"]

	if content != text {
		t.Errorf("expected content %q, got %v", text, content)
	}
}

func TestNewTableOfContents_EmptyColor(t *testing.T) {
	block := NewTableOfContents("")

	if block["type"] != "table_of_contents" {
		t.Errorf("expected type 'table_of_contents', got %v", block["type"])
	}

	toc := block["table_of_contents"].(map[string]interface{})
	if toc["color"] != "default" {
		t.Errorf("expected color 'default' for empty string input, got %v", toc["color"])
	}
}

func TestNewColumn(t *testing.T) {
	children := []map[string]interface{}{
		{"type": "paragraph", "paragraph": map[string]interface{}{"rich_text": []interface{}{}}},
	}

	block := NewColumn(children)

	if block["type"] != "column" {
		t.Errorf("expected type 'column', got %v", block["type"])
	}

	col := block["column"].(map[string]interface{})
	colChildren := col["children"].([]map[string]interface{})

	if len(colChildren) != 1 {
		t.Errorf("expected 1 child, got %d", len(colChildren))
	}

	if colChildren[0]["type"] != "paragraph" {
		t.Errorf("expected child type 'paragraph', got %v", colChildren[0]["type"])
	}
}

func TestNewTableRow(t *testing.T) {
	cells := [][]map[string]interface{}{
		{{"type": "text", "text": map[string]interface{}{"content": "A"}}},
		{{"type": "text", "text": map[string]interface{}{"content": "B"}}},
	}

	block := NewTableRow(cells)

	if block["type"] != "table_row" {
		t.Errorf("expected type 'table_row', got %v", block["type"])
	}

	tr := block["table_row"].(map[string]interface{})
	rowCells := tr["cells"].([][]map[string]interface{})
	if len(rowCells) != 2 {
		t.Errorf("expected 2 cells, got %d", len(rowCells))
	}
}

func TestNewTable(t *testing.T) {
	row1 := NewTableRow([][]map[string]interface{}{
		{{"type": "text", "text": map[string]interface{}{"content": "Name"}}},
		{{"type": "text", "text": map[string]interface{}{"content": "Role"}}},
	})
	row2 := NewTableRow([][]map[string]interface{}{
		{{"type": "text", "text": map[string]interface{}{"content": "Alice"}}},
		{{"type": "text", "text": map[string]interface{}{"content": "Eng"}}},
	})

	block := NewTable(2, true, []map[string]interface{}{row1, row2})

	if block["type"] != "table" {
		t.Errorf("expected type 'table', got %v", block["type"])
	}

	tbl := block["table"].(map[string]interface{})
	if tbl["table_width"] != 2 {
		t.Errorf("expected table_width 2, got %v", tbl["table_width"])
	}
	if tbl["has_column_header"] != true {
		t.Errorf("expected has_column_header true")
	}

	children := tbl["children"].([]map[string]interface{})
	if len(children) != 2 {
		t.Errorf("expected 2 children, got %d", len(children))
	}
	if children[0]["type"] != "table_row" {
		t.Errorf("expected child type 'table_row', got %v", children[0]["type"])
	}
}

func TestNewTableWithMarkdown(t *testing.T) {
	rows := [][]string{
		{"Name", "Role"},
		{"**Alice**", "Engineer"},
	}

	block := NewTableWithMarkdown(rows, true)

	if block["type"] != "table" {
		t.Errorf("expected type 'table', got %v", block["type"])
	}

	tbl := block["table"].(map[string]interface{})
	if tbl["table_width"] != 2 {
		t.Errorf("expected table_width 2, got %v", tbl["table_width"])
	}
	if tbl["has_column_header"] != true {
		t.Errorf("expected has_column_header true")
	}

	children := tbl["children"].([]map[string]interface{})
	if len(children) != 2 {
		t.Errorf("expected 2 rows, got %d", len(children))
	}

	// Check that inline markdown was parsed in the second row's first cell
	row2 := children[1]["table_row"].(map[string]interface{})
	cells := row2["cells"].([][]map[string]interface{})
	firstCell := cells[0]
	if len(firstCell) != 1 {
		t.Errorf("expected 1 rich_text element, got %d", len(firstCell))
	}
	if firstCell[0]["text"].(map[string]interface{})["content"] != "Alice" {
		t.Errorf("expected content 'Alice', got %v", firstCell[0]["text"])
	}
	ann := firstCell[0]["annotations"].(map[string]interface{})
	if ann["bold"] != true {
		t.Error("expected bold annotation on 'Alice'")
	}
}

func TestNewTableWithMarkdown_Empty(t *testing.T) {
	block := NewTableWithMarkdown(nil, false)
	tbl := block["table"].(map[string]interface{})
	if tbl["table_width"] != 0 {
		t.Errorf("expected table_width 0, got %v", tbl["table_width"])
	}
}

func TestBlockTypesSerialization(t *testing.T) {
	blocks := []map[string]interface{}{
		NewParagraph("test"),
		NewHeading1("test"),
		NewBreadcrumb(),
		NewDivider(),
	}

	for _, block := range blocks {
		data, err := json.Marshal(block)
		if err != nil {
			t.Errorf("failed to marshal block type %v: %v", block["type"], err)
		}
		if len(data) == 0 {
			t.Errorf("empty serialization for block type %v", block["type"])
		}
	}
}

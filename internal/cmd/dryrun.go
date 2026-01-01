package cmd

import (
	"fmt"
	"io"
)

// DryRunPrinter helps format dry-run output consistently across commands.
type DryRunPrinter struct {
	w io.Writer
}

// NewDryRunPrinter creates a new DryRunPrinter that writes to the given writer.
func NewDryRunPrinter(w io.Writer) *DryRunPrinter {
	return &DryRunPrinter{w: w}
}

// Header prints the header line indicating the action that would be taken.
// Example: [DRY-RUN] Would delete block abc-123
func (p *DryRunPrinter) Header(action, resourceType, id string) {
	_, _ = fmt.Fprintf(p.w, "[DRY-RUN] Would %s %s %s\n", action, resourceType, id)
}

// Field prints a single field with its value.
// Example:   Type: paragraph
func (p *DryRunPrinter) Field(name, value string) {
	_, _ = fmt.Fprintf(p.w, "  %s: %s\n", name, value)
}

// Change prints a field that would change from one value to another.
// Example:   Status: "In Progress" -> "Done"
func (p *DryRunPrinter) Change(name, oldVal, newVal string) {
	if oldVal == "" {
		_, _ = fmt.Fprintf(p.w, "  %s: (empty) -> %q\n", name, newVal)
	} else if newVal == "" {
		_, _ = fmt.Fprintf(p.w, "  %s: %q -> (empty)\n", name, oldVal)
	} else {
		_, _ = fmt.Fprintf(p.w, "  %s: %q -> %q\n", name, oldVal, newVal)
	}
}

// Unchanged prints a field that would remain unchanged.
// Example:   Priority: (unchanged)
func (p *DryRunPrinter) Unchanged(name string) {
	_, _ = fmt.Fprintf(p.w, "  %s: (unchanged)\n", name)
}

// Section prints a section header.
// Example:   Properties to update:
func (p *DryRunPrinter) Section(title string) {
	_, _ = fmt.Fprintf(p.w, "\n%s\n", title)
}

// Footer prints the footer message indicating no changes were made.
func (p *DryRunPrinter) Footer() {
	_, _ = fmt.Fprintf(p.w, "\n[DRY-RUN] No changes made.\n")
}

// Content prints multi-line content with proper indentation.
// Example:   Content: "This is the block content..."
func (p *DryRunPrinter) Content(name, value string) {
	_, _ = fmt.Fprintf(p.w, "  %s: %q\n", name, value)
}

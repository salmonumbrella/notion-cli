package cmd

// NotionMaxPageSize is the maximum page size allowed by the Notion API.
// All paginated endpoints limit page_size to 100.
const NotionMaxPageSize = 100

// capPageSize calculates the effective page size respecting both the user's
// limit and the Notion API maximum (100).
//
// The logic:
//   - If limit is 0 (unlimited), return pageSize unchanged
//   - If limit > 0 and pageSize is unset (0) or exceeds limit, cap it
//   - The cap is the lesser of limit and NotionMaxPageSize
//
// This ensures we never request more items than needed while respecting
// Notion's API constraints.
func capPageSize(pageSize, limit int) int {
	if limit > 0 && (pageSize == 0 || pageSize > limit) {
		if limit > NotionMaxPageSize {
			return NotionMaxPageSize
		}
		return limit
	}
	return pageSize
}

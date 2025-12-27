package cmd

type lightSearchResult struct {
	ID     string `json:"id"`
	Object string `json:"object,omitempty"`
	Title  string `json:"title,omitempty"`
	URL    string `json:"url,omitempty"`
}

func toLightSearchResults(results []map[string]interface{}) []lightSearchResult {
	light := make([]lightSearchResult, 0, len(results))
	for _, result := range results {
		if result == nil {
			continue
		}

		id, _ := result["id"].(string)
		if id == "" {
			continue
		}

		obj, _ := result["object"].(string)
		if obj == "data_source" {
			obj = "ds"
		}

		entry := lightSearchResult{
			ID:     id,
			Object: obj,
			Title:  extractResultTitle(result),
		}
		if url, ok := result["url"].(string); ok {
			entry.URL = url
		}

		light = append(light, entry)
	}

	return light
}

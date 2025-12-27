package cmd

import "github.com/salmonumbrella/notion-cli/internal/notion"

type lightPage struct {
	ID     string `json:"id"`
	Object string `json:"object,omitempty"`
	Title  string `json:"title,omitempty"`
	URL    string `json:"url,omitempty"`
}

func toLightPage(page *notion.Page) lightPage {
	if page == nil {
		return lightPage{}
	}

	objectType := page.Object
	if objectType == "" {
		objectType = "page"
	}

	return lightPage{
		ID:     page.ID,
		Object: objectType,
		Title:  extractPageTitleFromProperties(page.Properties),
		URL:    page.URL,
	}
}

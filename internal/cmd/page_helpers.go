package cmd

import (
	"context"
	"fmt"
	"strings"
)

func resolvePageParent(ctx context.Context, client databaseGetter, parentID, parentType, dataSourceID string) (map[string]interface{}, error) {
	if dataSourceID != "" {
		return map[string]interface{}{"data_source_id": dataSourceID}, nil
	}
	if parentID == "" {
		return nil, fmt.Errorf("parent is required")
	}

	switch strings.ToLower(strings.TrimSpace(parentType)) {
	case "page":
		return map[string]interface{}{"page_id": parentID}, nil
	case "database":
		resolved, err := resolveDataSourceID(ctx, client, parentID, "")
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"data_source_id": resolved}, nil
	case "data-source", "data_source", "datasource":
		return map[string]interface{}{"data_source_id": parentID}, nil
	default:
		return nil, fmt.Errorf("invalid parent-type: %s (expected 'page', 'database', or 'data-source')", parentType)
	}
}

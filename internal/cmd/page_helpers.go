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
	case "database", "db":
		resolved, err := resolveDataSourceID(ctx, client, parentID, "")
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"data_source_id": resolved}, nil
	case "data-source", "data_source", "datasource", "ds":
		return map[string]interface{}{"data_source_id": parentID}, nil
	default:
		return nil, fmt.Errorf("invalid parent-type: %s (expected 'page', 'database'/'db', or 'datasource'/'ds')", parentType)
	}
}

func resolveTitlePropertyNameForPageCreate(ctx context.Context, client pageSchemaGetter, parentID, parentType, dataSourceID string) (string, error) {
	if dataSourceID != "" {
		return titlePropertyNameFromDataSource(ctx, client, dataSourceID)
	}

	switch strings.ToLower(strings.TrimSpace(parentType)) {
	case "database", "db":
		db, err := client.GetDatabase(ctx, parentID)
		if err != nil {
			if hinted := maybeDataSourceHintForDatabaseNotFound(ctx, client, err, parentID); hinted != nil {
				return "", hinted
			}
			return "", fmt.Errorf("failed to get database schema for title property: %w", err)
		}
		return findTitlePropertyName(db.Properties), nil
	case "data-source", "data_source", "datasource", "ds":
		return titlePropertyNameFromDataSource(ctx, client, parentID)
	default:
		return "title", nil
	}
}

func titlePropertyNameFromDataSource(ctx context.Context, client pageSchemaGetter, dataSourceID string) (string, error) {
	ds, err := client.GetDataSource(ctx, dataSourceID)
	if err != nil {
		return "", fmt.Errorf("failed to get data source schema for title property: %w", err)
	}
	return findTitlePropertyNameFromDataSource(ds.Properties), nil
}

package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/salmonumbrella/notion-cli/internal/notion"
)

func resolveDataSourceID(ctx context.Context, client databaseGetter, databaseID string, dataSourceID string) (string, error) {
	if dataSourceID != "" {
		return dataSourceID, nil
	}
	if databaseID == "" {
		return "", fmt.Errorf("database ID is required to resolve data source")
	}

	database, err := client.GetDatabase(ctx, databaseID)
	if err != nil {
		if dsClient, ok := client.(dataSourceGetter); ok {
			if hinted := maybeDataSourceHintForDatabaseNotFound(ctx, dsClient, err, databaseID); hinted != nil {
				return "", hinted
			}
		}
		return "", fmt.Errorf("failed to get database: %w", err)
	}

	if len(database.DataSources) == 0 {
		return "", fmt.Errorf("database has no data sources")
	}
	if len(database.DataSources) > 1 {
		return "", fmt.Errorf("database has multiple data sources; specify --datasource (available: %s)", formatDataSourceChoices(database.DataSources))
	}

	return database.DataSources[0].ID, nil
}

func formatDataSourceChoices(dataSources []notion.DataSourceRef) string {
	parts := make([]string, 0, len(dataSources))
	for _, ds := range dataSources {
		if ds.Name != "" {
			parts = append(parts, fmt.Sprintf("%s (%s)", ds.Name, ds.ID))
		} else {
			parts = append(parts, ds.ID)
		}
	}
	return strings.Join(parts, ", ")
}

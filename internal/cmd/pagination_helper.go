package cmd

import "context"

// pageFetcher fetches a single page of results.
type pageFetcher[T any] func(ctx context.Context, cursor string, pageSize int) (results []T, nextCursor *string, hasMore bool, err error)

// fetchAllPages iterates over a paginated endpoint and returns aggregated results.
func fetchAllPages[T any](ctx context.Context, startCursor string, pageSize int, limit int, fetch pageFetcher[T]) ([]T, *string, bool, error) {
	cursor := startCursor
	var allResults []T
	var nextCursor *string
	hasMore := false

	for {
		results, next, more, err := fetch(ctx, cursor, pageSize)
		if err != nil {
			return nil, nextCursor, hasMore, err
		}

		allResults = append(allResults, results...)
		nextCursor = next
		hasMore = more

		if limit > 0 && len(allResults) >= limit {
			allResults = allResults[:limit]
			break
		}

		if !more || next == nil || *next == "" {
			break
		}
		cursor = *next
	}

	return allResults, nextCursor, hasMore, nil
}

var _ = fetchAllPages[struct{}]

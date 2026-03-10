package enrichment

import (
	"context"
	"log/slog"
	"sync"

	"go-sigil/internal/models"
)

// BatchEnrich enriches a slice of symbols concurrently.
// batchSize controls how many goroutines run at once.
// src maps symbol file path to raw source bytes.
// Errors are logged and don't stop the batch.
func BatchEnrich(ctx context.Context, enricher Enricher, symbols []*models.Symbol, src map[string][]byte, batchSize int) {
	if batchSize <= 0 {
		batchSize = 4
	}
	if len(symbols) == 0 {
		return
	}

	sem := make(chan struct{}, batchSize)
	var wg sync.WaitGroup

	for _, sym := range symbols {
		select {
		case <-ctx.Done():
			break
		default:
		}

		symSrc := src[sym.File]
		s := sym // capture
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			if err := enricher.Enrich(ctx, s, symSrc); err != nil {
				slog.Warn("enrich symbol failed", "symbol", s.QualifiedName, "error", err)
			}
		}()
	}
	wg.Wait()
}

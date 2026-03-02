package indexer

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"sort"
	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/release"
	"strings"
	"sync"
)

type Aggregator struct {
	Indexers []Indexer
}

func (a *Aggregator) Name() string {
	return "Aggregator"
}

func (a *Aggregator) GetIndexers() []Indexer {
	return a.Indexers
}

func (a *Aggregator) GetUsage() Usage {
	var usage Usage
	for _, idx := range a.Indexers {
		u := idx.GetUsage()
		usage.APIHitsLimit += u.APIHitsLimit
		usage.APIHitsUsed += u.APIHitsUsed
		usage.APIHitsRemaining += u.APIHitsRemaining
		usage.DownloadsLimit += u.DownloadsLimit
		usage.DownloadsUsed += u.DownloadsUsed
		usage.DownloadsRemaining += u.DownloadsRemaining
		usage.AllTimeAPIHitsUsed += u.AllTimeAPIHitsUsed
		usage.AllTimeDownloadsUsed += u.AllTimeDownloadsUsed
	}
	return usage
}

func NewAggregator(indexers ...Indexer) *Aggregator {
	return &Aggregator{
		Indexers: indexers,
	}
}

func (a *Aggregator) Ping() error {
	var lastErr error
	successCount := 0

	for _, idx := range a.Indexers {
		if err := idx.Ping(); err != nil {
			lastErr = err
		} else {
			successCount++
		}
	}

	if successCount == 0 && len(a.Indexers) > 0 {
		return fmt.Errorf("all indexers failed ping, last error: %w", lastErr)
	}
	return nil
}

func (a *Aggregator) DownloadNZB(ctx context.Context, nzbURL string) ([]byte, error) {
	if len(a.Indexers) == 0 {
		return nil, fmt.Errorf("no indexers configured")
	}
	var lastErr error
	for _, idx := range a.Indexers {
		data, err := idx.DownloadNZB(ctx, nzbURL)
		if err == nil {
			return data, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (a *Aggregator) ResolveDownloadURL(ctx context.Context, directURL, title string, size int64, cat string) (string, error) {
	if title == "" {
		return "", fmt.Errorf("title required to resolve download URL")
	}
	req := SearchRequest{Query: title, Limit: 30, Cat: cat}
	resp, err := a.Search(req)
	if err != nil {
		return "", fmt.Errorf("search for resolve: %w", err)
	}
	if resp == nil || len(resp.Channel.Items) == 0 {
		return "", fmt.Errorf("no search results for title")
	}
	normTitle := release.NormalizeTitle(title)
	var bestMatch string
	for _, item := range resp.Channel.Items {
		if release.NormalizeTitle(item.Title) != normTitle {
			continue
		}
		if item.Link == "" {
			continue
		}

		if size > 0 && item.Size > 0 && item.Size == size {
			return item.Link, nil
		}

		if bestMatch == "" {
			bestMatch = item.Link
		}
	}
	if bestMatch != "" {
		return bestMatch, nil
	}
	return "", fmt.Errorf("no matching release for title in search results")
}

func (a *Aggregator) Search(req SearchRequest) (*SearchResponse, error) {
	resultsChan := make(chan []Item, len(a.Indexers))
	var wg sync.WaitGroup

	for _, idx := range a.Indexers {
		wg.Add(1)
		queries := req.PerIndexerQuery[idx.Name()]
		if req.PerIndexerQuery != nil && len(queries) == 0 {
			wg.Done()
			resultsChan <- []Item{}
			continue
		}
		if req.PerIndexerQuery != nil && len(queries) > 0 {

			go func(indexer Indexer) {
				defer wg.Done()
				var merged []Item
				for _, q := range queries {
					if q == "" {
						continue
					}
					reqCopy := req
					reqCopy.EffectiveByIndexer = nil
					reqCopy.PerIndexerQuery = nil
					if req.EffectiveByIndexer != nil {
						reqCopy.OptionalOverrides = req.EffectiveByIndexer[indexer.Name()]
					}
					reqCopy.Query = q
					resp, err := indexer.Search(reqCopy)
					if err != nil {
						logger.Warn("Indexer search failed", "indexer", indexer.Name(), "query", q, "err", err)
						continue
					}
					if resp != nil && len(resp.Channel.Items) > 0 {
						merged = append(merged, resp.Channel.Items...)
					}
				}
				resultsChan <- merged
			}(idx)
			continue
		}
		reqCopy := req
		reqCopy.EffectiveByIndexer = nil
		reqCopy.PerIndexerQuery = nil
		if req.EffectiveByIndexer != nil {
			reqCopy.OptionalOverrides = req.EffectiveByIndexer[idx.Name()]
		}
		if req.Query != "" {
			reqCopy.Query = req.Query
		}
		go func(indexer Indexer, r SearchRequest) {
			defer wg.Done()

			resp, err := indexer.Search(r)
			if err != nil {
				logger.Warn("Indexer search failed", "indexer", indexer.Name(), "err", err)
				resultsChan <- []Item{}
				return
			}

			if resp != nil {
				resultsChan <- resp.Channel.Items
			} else {
				resultsChan <- []Item{}
			}
		}(idx, reqCopy)
	}

	wg.Wait()
	close(resultsChan)

	var allItems []Item
	for items := range resultsChan {
		allItems = append(allItems, items...)
	}

	seenGUID := make(map[string]bool)
	seenLink := make(map[string]bool)
	seenTitleSize := make(map[string]bool)
	uniqueItems := []Item{}

	for _, item := range allItems {

		normalizedTitle := release.NormalizeTitleForDedup(item.Title)

		if item.GUID != "" {
			if seenGUID[item.GUID] {
				continue
			}
			seenGUID[item.GUID] = true
			uniqueItems = append(uniqueItems, item)
			continue
		}

		if item.Link != "" {

			normalizedLink := normalizeURL(item.Link)
			if seenLink[normalizedLink] {
				continue
			}
			seenLink[normalizedLink] = true
			uniqueItems = append(uniqueItems, item)
			continue
		}

		titleSizeKey := fmt.Sprintf("%s:%d", normalizedTitle, item.Size)
		if item.Size > 0 && seenTitleSize[titleSizeKey] {
			continue
		}
		if item.Size > 0 {
			seenTitleSize[titleSizeKey] = true
		}
		uniqueItems = append(uniqueItems, item)
	}

	sort.Slice(uniqueItems, func(i, j int) bool {
		return uniqueItems[i].Size > uniqueItems[j].Size
	})

	resp := &SearchResponse{
		XMLName: xml.Name{Local: "rss"},
		Channel: Channel{
			Items: uniqueItems,
		},
	}
	NormalizeSearchResponse(resp)
	return resp, nil
}

func normalizeURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(rawURL))
	}

	normalized := fmt.Sprintf("%s://%s%s", parsed.Scheme, parsed.Host, parsed.Path)
	return strings.ToLower(strings.TrimSpace(normalized))
}

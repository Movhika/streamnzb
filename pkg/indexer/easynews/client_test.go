package easynews

import (
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/core/persistence"
	"streamnzb/pkg/indexer"
)

func init() {
	logger.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
}

var (
	easynewsUsageManagerOnce sync.Once
	easynewsUsageManager     *indexer.UsageManager
	easynewsUsageManagerErr  error
)

func testEasynewsUsageManager(t *testing.T) *indexer.UsageManager {
	t.Helper()

	easynewsUsageManagerOnce.Do(func() {
		tempDir, err := os.MkdirTemp("", "streamnzb-easynews-usage-")
		if err != nil {
			easynewsUsageManagerErr = err
			return
		}
		stateMgr, err := persistence.GetManager(tempDir)
		if err != nil {
			easynewsUsageManagerErr = err
			return
		}
		easynewsUsageManager, easynewsUsageManagerErr = indexer.GetUsageManager(stateMgr)
	})
	if easynewsUsageManagerErr != nil {
		t.Fatalf("GetUsageManager: %v", easynewsUsageManagerErr)
	}
	return easynewsUsageManager
}

func TestGetUsageRefreshesDailyCountersAfterRollover(t *testing.T) {
	usageManager := testEasynewsUsageManager(t)
	name := "easynews-rollover-usage"
	usageData := usageManager.GetIndexerUsage(name)
	usageData.LastResetDay = time.Now().Format("2006-01-02")
	usageData.APIHitsUsed = 8
	usageData.DownloadsUsed = 4
	usageData.AllTimeAPIHitsUsed = 30
	usageData.AllTimeDownloadsUsed = 12

	client, err := NewClient("user", "pass", name, "", 8, 4, 0, usageManager)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	usageData.LastResetDay = time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	usageData.APIHitsUsed = 8
	usageData.DownloadsUsed = 4

	usage := client.GetUsage()
	if usage.APIHitsUsed != 0 || usage.DownloadsUsed != 0 {
		t.Fatalf("expected refreshed daily usage to reset, got hits=%d downloads=%d", usage.APIHitsUsed, usage.DownloadsUsed)
	}
	if usage.APIHitsRemaining != 8 || usage.DownloadsRemaining != 4 {
		t.Fatalf("expected refreshed remaining counts, got api=%d downloads=%d", usage.APIHitsRemaining, usage.DownloadsRemaining)
	}
	if usage.AllTimeAPIHitsUsed != 30 || usage.AllTimeDownloadsUsed != 12 {
		t.Fatalf("expected all-time usage unchanged, got hits=%d downloads=%d", usage.AllTimeAPIHitsUsed, usage.AllTimeDownloadsUsed)
	}
}

func TestLimitChecksRefreshDailyUsageAfterRollover(t *testing.T) {
	usageManager := testEasynewsUsageManager(t)
	name := "easynews-rollover-limits"
	usageData := usageManager.GetIndexerUsage(name)
	usageData.LastResetDay = time.Now().Format("2006-01-02")
	usageData.APIHitsUsed = 8
	usageData.DownloadsUsed = 4

	client, err := NewClient("user", "pass", name, "", 8, 4, 0, usageManager)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	usageData.LastResetDay = time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	usageData.APIHitsUsed = 8
	usageData.DownloadsUsed = 4
	usageData.AllTimeAPIHitsUsed = 35
	usageData.AllTimeDownloadsUsed = 14

	if err := client.checkAPILimit(); err != nil {
		t.Fatalf("checkAPILimit() error = %v, want nil after rollover refresh", err)
	}
	if err := client.checkDownloadLimit(); err != nil {
		t.Fatalf("checkDownloadLimit() error = %v, want nil after rollover refresh", err)
	}
}

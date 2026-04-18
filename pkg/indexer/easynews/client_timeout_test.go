package easynews

import "testing"

func TestNewClientConfiguresSeparateSearchAndDownloadTimeouts(t *testing.T) {
	client, err := NewClient("user", "pass", "Easynews", "", 0, 0, 0, nil)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	if client.client == nil {
		t.Fatal("expected search client to be configured")
	}
	if client.downloadClient == nil {
		t.Fatal("expected download client to be configured")
	}
	if client.client.Timeout != searchTimeout {
		t.Fatalf("expected search timeout %v, got %v", searchTimeout, client.client.Timeout)
	}
	if client.downloadClient.Timeout != downloadTimeout {
		t.Fatalf("expected download timeout %v, got %v", downloadTimeout, client.downloadClient.Timeout)
	}
}

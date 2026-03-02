package nzb

import (
	"os"
	"path/filepath"
	"testing"

	"streamnzb/pkg/core/logger"
)

func TestCompressionType_posterAttribute(t *testing.T) {

	logger.Init("warn")

	path := filepath.Join("..", "..", "..", "The.Hobbit.The.Desolation.Of.Smaug.Extended.(2013).HDR.10bit.2160p.BT2020.DTS.HD.MA-VISIONPLUSHDR1000.NLsubs.nzb")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("NZB file not found: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()
	nzb, err := Parse(f)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	ct := nzb.CompressionType()
	if ct != "rar" {
		t.Errorf("CompressionType() = %q, want %q", ct, "rar")
	}
	contentFiles := nzb.GetContentFiles()
	if len(contentFiles) == 0 {
		t.Error("GetContentFiles() returned empty, expected RAR parts")
	}
}

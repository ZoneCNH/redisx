package goalruntime

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCommandsReturnsStableCommandList(t *testing.T) {
	want := []string{
		"goal-acceptance",
		"goal-delivery",
		"goal-handover",
		"goal-downstream-adoption",
		"goal-certify",
		"goal-runtime-final",
	}
	got := Commands()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Commands() = %#v; want %#v", got, want)
	}
	got[0] = "changed"
	if Commands()[0] == "changed" {
		t.Fatal("Commands returned a mutable shared slice")
	}
}

func TestModulePathForRootBranches(t *testing.T) {
	root := t.TempDir()
	if _, ok := modulePathForRoot(root); ok {
		t.Fatal("modulePathForRoot found module in missing go.mod")
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("go 1.25\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if _, ok := modulePathForRoot(root); ok {
		t.Fatal("modulePathForRoot found module in go.mod without module line")
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/ZoneCNH/redisx\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatalf("rewrite go.mod: %v", err)
	}
	if got, ok := modulePathForRoot(root); !ok || got != "github.com/ZoneCNH/redisx" {
		t.Fatalf("modulePathForRoot = %q, %t; want redisx module", got, ok)
	}
}

func TestReadLedgerEntriesHandlesBlankLinesAndInvalidJSON(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "ledger.jsonl")
	if err := os.WriteFile(path, []byte("\n{\"schema_version\":\"goal-runtime-evidence/v1\",\"goal_id\":\"G\",\"command\":\"goal-acceptance\"}\n\n"), 0o644); err != nil {
		t.Fatalf("write ledger: %v", err)
	}
	entries, err := readLedgerEntries(path)
	if err != nil {
		t.Fatalf("readLedgerEntries returned error: %v", err)
	}
	if len(entries) != 1 || entries[0].Command != "goal-acceptance" {
		t.Fatalf("entries = %#v; want one parsed entry", entries)
	}

	badPath := filepath.Join(root, "bad.jsonl")
	if err := os.WriteFile(badPath, []byte("{bad}\n"), 0o644); err != nil {
		t.Fatalf("write bad ledger: %v", err)
	}
	if _, err := readLedgerEntries(badPath); err == nil || !strings.Contains(err.Error(), "invalid ledger entry on line 1") {
		t.Fatalf("readLedgerEntries bad error = %v; want line-specific parse error", err)
	}
}

func TestWriteEvidenceRejectsUnsupportedAndIncompleteReports(t *testing.T) {
	root := t.TempDir()
	err := WriteEvidence(root, Report{
		Command:          "not-a-command",
		Status:           "passed",
		MVAStatus:        "complete",
		Blocking:         true,
		GoalID:           DefaultGoalID,
		LedgerPath:       SourceLedgerPath,
		EvidencePackPath: EvidenceLedgerPath + "bad.json",
	})
	if err == nil || !strings.Contains(err.Error(), "evidence write is not supported") {
		t.Fatalf("WriteEvidence unsupported error = %v; want unsupported command", err)
	}

	err = WriteEvidence(root, Report{
		Command:          "goal-acceptance",
		Status:           "failed",
		MVAStatus:        "not-complete",
		Blocking:         false,
		GoalID:           DefaultGoalID,
		LedgerPath:       SourceLedgerPath,
		EvidencePackPath: EvidenceLedgerPath + "incomplete.json",
	})
	if err == nil || !strings.Contains(err.Error(), "refuse to write incomplete goalcli evidence") {
		t.Fatalf("WriteEvidence incomplete error = %v; want incomplete report refusal", err)
	}
}

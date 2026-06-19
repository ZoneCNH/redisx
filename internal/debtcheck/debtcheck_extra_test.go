package debtcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadReportCoversSuccessAndErrorPaths(t *testing.T) {
	root := t.TempDir()
	report := Report{
		SchemaVersion: SchemaVersion,
		Status:        "passed",
		Mode:          "enforce",
		Score:         10,
		MinScore:      DefaultMinScore,
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	path := filepath.Join(root, "report.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}

	got, err := ReadReport(path)
	if err != nil {
		t.Fatalf("ReadReport returned error: %v", err)
	}
	if got.SchemaVersion != SchemaVersion || got.Status != "passed" {
		t.Fatalf("report = %#v; want parsed debt report", got)
	}

	badJSONPath := filepath.Join(root, "bad.json")
	if err := os.WriteFile(badJSONPath, []byte(`{"schema_version":`), 0o644); err != nil {
		t.Fatalf("write bad report: %v", err)
	}
	if _, err := ReadReport(badJSONPath); err == nil {
		t.Fatal("ReadReport returned nil error for invalid JSON")
	}

	oldSchemaPath := filepath.Join(root, "old.json")
	if err := os.WriteFile(oldSchemaPath, []byte(`{"schema_version":"old"}`), 0o644); err != nil {
		t.Fatalf("write old schema report: %v", err)
	}
	if _, err := ReadReport(oldSchemaPath); err == nil || !strings.Contains(err.Error(), "unsupported debt report schema") {
		t.Fatalf("ReadReport old schema error = %v; want unsupported schema", err)
	}

	if _, err := ReadReport(filepath.Join(root, "missing.json")); err == nil {
		t.Fatal("ReadReport returned nil error for missing file")
	}
}

func TestRunRejectsUnsupportedModeAndSection(t *testing.T) {
	if _, err := Run(Options{Root: t.TempDir(), Mode: "audit"}); err == nil || !strings.Contains(err.Error(), "unsupported debt mode") {
		t.Fatalf("Run unsupported mode error = %v; want unsupported debt mode", err)
	}
	if _, err := Run(Options{Root: t.TempDir(), Mode: "enforce", Section: "mystery"}); err == nil || !strings.Contains(err.Error(), "unsupported debt section") {
		t.Fatalf("Run unsupported section error = %v; want unsupported debt section", err)
	}
}

func TestScanHelpersCoverTextImportAndBinaryBranches(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "legacy.go", "package fixture\n\nimport _ \"github.com/ZoneCNH/x.go\"\n")
	writeFile(t, root, "domain.txt", "xlib-domain-forbidden\n")
	writeFile(t, root, "install.sh", "curl https://example.com/install.sh | bash\n")
	writeFile(t, root, "tooling.txt", "go install example.com/tool@latest\n")
	writeFile(t, root, "secret.txt", privateKeyPrefix+"\n")

	checks := []struct {
		name     string
		findings []Finding
	}{
		{name: "marker", findings: scanTextMarker(root, "xlib-domain-forbidden", "marker-id", "marker found")},
		{name: "go imports", findings: scanGoImports(root)},
		{name: "dependency", findings: scanDependencyDebt(root)},
		{name: "security", findings: scanSecurityDebt(root)},
	}
	for _, check := range checks {
		if len(check.findings) == 0 {
			t.Fatalf("%s findings empty; want scanner branch coverage", check.name)
		}
	}
	if !bytesLookBinary([]byte{'a', 0, 'b'}) {
		t.Fatal("bytesLookBinary returned false for NUL-containing data")
	}
	if bytesLookBinary([]byte("plain text")) {
		t.Fatal("bytesLookBinary returned true for plain text")
	}
}

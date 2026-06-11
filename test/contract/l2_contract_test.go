package contract

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type readinessEvidence struct {
	Profile string `json:"profile"`
	Path    string `json:"path"`
	Status  string `json:"status"`
}

type releaseReadiness struct {
	SchemaVersion string              `json:"schema_version"`
	Adapter       string              `json:"adapter"`
	TargetLevel   string              `json:"target_level"`
	Score         int                 `json:"score"`
	Profiles      []string            `json:"profiles"`
	Evidence      []readinessEvidence `json:"evidence"`
}

func TestL2ContractPackDeclaration(t *testing.T) {
	manifest, err := os.ReadFile("../../.agent/l2-capabilities.yaml")
	if err != nil {
		t.Fatalf("read L2 capability manifest: %v", err)
	}

	text := string(manifest)
	requiredSnippets := []string{
		`schema_version: "1.0"`,
		"layer: L2",
		"name: redisx",
		"module: github.com/ZoneCNH/redisx",
		"family: key_value",
		"contract_packs:",
		"- common",
		"- kv",
		"- ttl",
		"- pool",
		"required_profiles:",
		"- unit",
		"- contract",
		"- integration",
		"output_dir: .agent/evidence/l2",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(text, snippet) {
			t.Fatalf("L2 manifest missing required snippet %q", snippet)
		}
	}

	forbiddenKeys := []string{
		"provider_endpoint:",
		"provider_credentials:",
		"password:",
		"secret:",
		"token:",
	}
	for _, key := range forbiddenKeys {
		if strings.Contains(strings.ToLower(text), key) {
			t.Fatalf("L2 manifest contains forbidden provider-boundary key %q", key)
		}
	}
}

func TestL2ReleaseReadinessSnapshot(t *testing.T) {
	raw, err := os.ReadFile("../../.agent/evidence/l2/release-readiness.json")
	if err != nil {
		t.Fatalf("read L2 release readiness: %v", err)
	}

	var readiness releaseReadiness
	if err := json.Unmarshal(raw, &readiness); err != nil {
		t.Fatalf("parse L2 release readiness: %v", err)
	}

	if readiness.SchemaVersion != "1.0" {
		t.Fatalf("unexpected schema version %q", readiness.SchemaVersion)
	}
	if readiness.Adapter != "redisx" {
		t.Fatalf("unexpected adapter %q", readiness.Adapter)
	}
	if readiness.TargetLevel != "L2-T2" {
		t.Fatalf("unexpected target level %q", readiness.TargetLevel)
	}
	for _, profile := range []string{"unit", "contract", "integration"} {
		if !contains(readiness.Profiles, profile) {
			t.Fatalf("readiness missing profile %q", profile)
		}
	}

	statuses := map[string]string{}
	for _, evidence := range readiness.Evidence {
		statuses[evidence.Profile] = evidence.Status
	}
	if statuses["unit"] != "pass" {
		t.Fatalf("unit evidence status = %q, want pass", statuses["unit"])
	}
	if statuses["contract"] != "missing" {
		t.Fatalf("contract evidence status = %q, want missing until contract report exists", statuses["contract"])
	}
	if statuses["integration"] != "missing" {
		t.Fatalf("integration evidence status = %q, want missing until integration report exists", statuses["integration"])
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

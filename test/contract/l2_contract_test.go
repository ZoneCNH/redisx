package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	ReleaseReady  bool                `json:"release_ready"`
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
	if readiness.Score != 100 {
		t.Fatalf("readiness score = %d, want 100", readiness.Score)
	}
	if !readiness.ReleaseReady {
		t.Fatal("readiness release_ready = false, want true")
	}
	for _, profile := range []string{"unit", "contract", "integration"} {
		if !contains(readiness.Profiles, profile) {
			t.Fatalf("readiness missing profile %q", profile)
		}
	}

	statuses := map[string]string{}
	for _, evidence := range readiness.Evidence {
		statuses[evidence.Profile] = evidence.Status
		if strings.HasPrefix(evidence.Path, ".agent/") {
			path := filepath.Clean(filepath.Join("../..", evidence.Path))
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("readiness evidence path %q is not available: %v", evidence.Path, err)
			}
		}
	}
	for _, profile := range []string{"unit", "contract", "integration"} {
		if statuses[profile] != "pass" {
			t.Fatalf("%s evidence status = %q, want pass", profile, statuses[profile])
		}
	}
}

func TestL2DevRedisEndpointConfig(t *testing.T) {
	compose, err := os.ReadFile("../../docker-compose.yml")
	if err != nil {
		t.Fatalf("read docker compose config: %v", err)
	}
	devcontainer, err := os.ReadFile("../../.devcontainer/devcontainer.json")
	if err != nil {
		t.Fatalf("read devcontainer config: %v", err)
	}

	composeText := string(compose)
	devcontainerText := string(devcontainer)
	requiredComposeSnippets := []string{
		"image: redis:7.2-alpine",
		"REDISX_REDIS_ADDR: ${REDISX_REDIS_ADDR:-redis:6379}",
		"REDISX_REDIS_URL: ${REDISX_REDIS_URL:-redis://redis:6379/0}",
		"REDISX_REDIS_DB: ${REDISX_REDIS_DB:-0}",
	}
	for _, snippet := range requiredComposeSnippets {
		if !strings.Contains(composeText, snippet) {
			t.Fatalf("docker-compose missing required Redis endpoint snippet %q", snippet)
		}
	}

	requiredDevcontainerSnippets := []string{
		`"REDISX_REDIS_ADDR": "redis:6379"`,
		`"REDISX_REDIS_URL": "redis://redis:6379/0"`,
		`"REDISX_REDIS_DB": "0"`,
	}
	for _, snippet := range requiredDevcontainerSnippets {
		if !strings.Contains(devcontainerText, snippet) {
			t.Fatalf("devcontainer missing required Redis endpoint snippet %q", snippet)
		}
	}

	combined := strings.ToLower(composeText + "\n" + devcontainerText)
	for _, forbidden := range []string{"REDISX_REDIS_PASSWORD", "REDISX_REDIS_TOKEN", "REDISX_REDIS_SECRET"} {
		if strings.Contains(combined, strings.ToLower(forbidden)) {
			t.Fatalf("dev Redis endpoint config exposes forbidden secret env var %q", forbidden)
		}
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

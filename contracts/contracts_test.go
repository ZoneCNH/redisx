package contracts

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ZoneCNH/redisx/pkg/redisx"
)

type schemaProperty struct {
	Type    string   `json:"type"`
	Enum    []string `json:"enum"`
	Minimum *int     `json:"minimum"`
}

type objectSchema struct {
	Required   []string                  `json:"required"`
	Properties map[string]schemaProperty `json:"properties"`
}

func TestErrorKindContractMatchesPublicConstants(t *testing.T) {
	schema := readSchema(t, "error.schema.json")

	expected := sortedStrings(
		string(redisx.ErrorKindConfig),
		string(redisx.ErrorKindValidation),
		string(redisx.ErrorKindConnection),
		string(redisx.ErrorKindUnavailable),
		string(redisx.ErrorKindTimeout),
		string(redisx.ErrorKindAuth),
		string(redisx.ErrorKindNetwork),
		string(redisx.ErrorKindReadOnly),
		string(redisx.ErrorKindLoading),
		string(redisx.ErrorKindTryAgain),
		string(redisx.ErrorKindClusterMoved),
		string(redisx.ErrorKindClusterAsk),
		string(redisx.ErrorKindConflict),
		string(redisx.ErrorKindRateLimit),
		string(redisx.ErrorKindInternal),
		string(redisx.ErrorKindCanceled),
		string(redisx.ErrorKindNil),
		string(redisx.ErrorKindClosed),
		string(redisx.ErrorKindInvalidConfig),
		string(redisx.ErrorKindProvider),
	)
	actual := sortedStrings(schema.Properties["kind"].Enum...)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("error kind contract drift:\nactual:   %#v\nexpected: %#v", actual, expected)
	}
	requireFields(t, schema.Required, "kind", "op", "message", "retryable")
}

func TestHealthStatusContractMatchesPublicConstants(t *testing.T) {
	schema := readSchema(t, "health.schema.json")

	expected := sortedStrings(
		string(redisx.HealthHealthy),
		string(redisx.HealthDegraded),
		string(redisx.HealthUnhealthy),
	)
	actual := sortedStrings(schema.Properties["status"].Enum...)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("health status contract drift:\nactual:   %#v\nexpected: %#v", actual, expected)
	}
	requireFields(t, schema.Required, "name", "status", "checked_at")
}

func TestConfigContractMatchesPublicConfig(t *testing.T) {
	schema := readSchema(t, "config.schema.json")
	requireFields(t, schema.Required, "name")

	configType := reflect.TypeOf(redisx.Config{})
	requireSchemaFieldMapsToStructField(t, schema, configType, "name", "Name", "string")
	requireSchemaFieldMapsToStructField(t, schema, configType, "timeout_ms", "Timeout", "integer")
	requireSchemaFieldMapsToStructField(t, schema, configType, "secret", "Secret", "string")
	requireSchemaFieldMapsToStructField(t, schema, configType, "redis", "Redis", "object")

	if timeoutField, ok := configType.FieldByName("Timeout"); !ok || timeoutField.Type != reflect.TypeOf(time.Duration(0)) {
		t.Fatalf("Config.Timeout must remain time.Duration, got %v", timeoutField.Type)
	}
	if redisField, ok := configType.FieldByName("Redis"); !ok || redisField.Type != reflect.TypeOf(redisx.RedisConfig{}) {
		t.Fatalf("Config.Redis must remain redisx.RedisConfig, got %v", redisField.Type)
	}
	if minimum := schema.Properties["timeout_ms"].Minimum; minimum == nil || *minimum != 0 {
		t.Fatalf("timeout_ms must define minimum 0, got %#v", minimum)
	}
	text := readText(t, "config.schema.json")
	for _, needle := range redisConfigSchemaMarkers() {
		if !strings.Contains(text, needle) {
			t.Fatalf("config contract missing marker %s", needle)
		}
	}
}

func TestMetricsContractDocumentsPublicConstants(t *testing.T) {
	content, err := os.ReadFile("metrics.md")
	if err != nil {
		t.Fatalf("read metrics contract: %v", err)
	}
	text := string(content)
	for _, metric := range []string{
		redisx.MetricClientCreatedTotal,
		redisx.MetricClientClosedTotal,
		redisx.MetricClientErrorsTotal,
		redisx.MetricClientHealthStatus,
		redisx.MetricClientHealthLatencyMS,
		redisx.MetricClientRequestsTotal,
		redisx.MetricClientRequestDurationSeconds,
		redisx.MetricClientRetriesTotal,
		redisx.MetricClientInflight,
		redisx.MetricRedisOperationsTotal,
		redisx.MetricRedisOperationDurationSeconds,
		redisx.MetricRedisErrorsTotal,
		redisx.MetricRedisPoolConnections,
		redisx.MetricRedisHealthStatus,
	} {
		if !strings.Contains(text, "`"+metric+"`") {
			t.Fatalf("metrics contract does not document %q", metric)
		}
	}
}

func TestRedisxConfigContractMatchesPublicOptions(t *testing.T) {
	schema := readSchema(t, "redisx.config.schema.json")
	requireFields(t, schema.Required, "config")
	if got := schema.Properties["config"].Type; got != "object" {
		t.Fatalf("redisx config schema config type = %q, want object", got)
	}

	optionsType := reflect.TypeOf(redisx.Options{})
	for _, field := range []string{"Config", "Metrics", "Provider"} {
		if _, ok := optionsType.FieldByName(field); !ok {
			t.Fatalf("redisx.Options missing field %s required by contract", field)
		}
	}

	text := readText(t, "redisx.config.schema.json")
	for _, needle := range append([]string{"\"name\"", "\"timeout_ms\"", "\"secret\"", "\"in_memory\"", "\"redis\"", "\"custom\""}, redisConfigSchemaMarkers()...) {
		if !strings.Contains(text, needle) {
			t.Fatalf("redisx config contract missing marker %s", needle)
		}
	}
}

func redisConfigSchemaMarkers() []string {
	return []string{
		"\"addr\"",
		"\"username\"",
		"\"password\"",
		"\"db\"",
		"\"dial_timeout_ms\"",
		"\"read_timeout_ms\"",
		"\"write_timeout_ms\"",
		"\"pool_size\"",
		"\"min_idle_conns\"",
		"\"max_retries\"",
	}
}

func TestRedisxHealthContractMatchesPublicStatus(t *testing.T) {
	schema := readSchema(t, "redisx.health.schema.json")
	expected := sortedStrings(
		string(redisx.HealthHealthy),
		string(redisx.HealthDegraded),
		string(redisx.HealthUnhealthy),
	)
	actual := sortedStrings(schema.Properties["status"].Enum...)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("redisx health status contract drift:\nactual:   %#v\nexpected: %#v", actual, expected)
	}
	requireFields(t, schema.Required, "name", "component", "status", "checked_at", "latency_ms")
	for _, field := range []string{"message", "error_class", "metadata"} {
		if _, ok := schema.Properties[field]; !ok {
			t.Fatalf("redisx health schema missing property %q", field)
		}
	}
}

func TestRedisxErrorTaxonomyContractDocumentsPublicIdentifiers(t *testing.T) {
	text := readText(t, "redisx.errors.yaml")
	for _, item := range []struct {
		name string
		id   redisx.RedisErrorID
	}{
		{"ErrNil", redisx.ErrNil},
		{"ErrTimeout", redisx.ErrTimeout},
		{"ErrCanceled", redisx.ErrCanceled},
		{"ErrNetwork", redisx.ErrNetwork},
		{"ErrAuth", redisx.ErrAuth},
		{"ErrReadOnly", redisx.ErrReadOnly},
		{"ErrLoading", redisx.ErrLoading},
		{"ErrTryAgain", redisx.ErrTryAgain},
		{"ErrClusterMoved", redisx.ErrClusterMoved},
		{"ErrClusterAsk", redisx.ErrClusterAsk},
		{"ErrConnectionClosed", redisx.ErrConnectionClosed},
		{"ErrInvalidConfig", redisx.ErrInvalidConfig},
		{"ErrProvider", redisx.ErrProvider},
	} {
		for _, needle := range []string{item.name, item.id.String(), string(item.id.Kind())} {
			if !strings.Contains(text, needle) {
				t.Fatalf("redisx error contract missing %q for %s", needle, item.name)
			}
		}
	}
}

func TestRedisxMetricsContractDocumentsPublicConstants(t *testing.T) {
	text := readText(t, "redisx.metrics.yaml")
	for _, metric := range []string{
		redisx.MetricClientCreatedTotal,
		redisx.MetricClientClosedTotal,
		redisx.MetricClientErrorsTotal,
		redisx.MetricClientHealthStatus,
		redisx.MetricClientHealthLatencyMS,
		redisx.MetricClientRequestsTotal,
		redisx.MetricClientRequestDurationSeconds,
		redisx.MetricClientRetriesTotal,
		redisx.MetricClientInflight,
		redisx.MetricRedisOperationsTotal,
		redisx.MetricRedisOperationDurationSeconds,
		redisx.MetricRedisErrorsTotal,
		redisx.MetricRedisPoolConnections,
		redisx.MetricRedisHealthStatus,
	} {
		if !strings.Contains(text, metric) {
			t.Fatalf("redisx metrics contract does not document %q", metric)
		}
	}
}

func TestGoalRuntimeSchemasAreValidJSON(t *testing.T) {
	for _, path := range []string{
		"goalcli-report.schema.json",
		"goalcli-dashboard.schema.json",
		"issue-registry.schema.json",
		"command-registry.schema.json",
		"layer-governance.schema.json",
		"execution-context.schema.json",
		"conformance-attestation.schema.json",
		"policy.schema.json",
		"execution-evidence.schema.json",
		"downstream-adoption-proof.schema.json",
		"docker-toolchain.schema.json",
		"redisx.config.schema.json",
		"redisx.health.schema.json",
	} {
		t.Run(path, func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			var schema map[string]any
			if err := json.Unmarshal(content, &schema); err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			if schema["$schema"] == "" || schema["type"] != "object" {
				t.Fatalf("%s must declare object JSON schema, got %#v", path, schema)
			}
		})
	}
}

func TestExecutionContextContractMatchesGovernanceContexts(t *testing.T) {
	schema := readSchema(t, "execution-context.schema.json")

	expected := sortedStrings("local_write", "local_readonly", "ci_pull_request", "ci_main_verify", "release_verify")
	actual := sortedStrings(schema.Properties["context"].Enum...)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("execution context enum drift:\nactual:   %#v\nexpected: %#v", actual, expected)
	}
	requireFields(t, schema.Required, "context", "root", "gowork")
}

func TestExecutionEvidenceContractRequiredFields(t *testing.T) {
	schema := readSchema(t, "execution-evidence.schema.json")
	requireFields(t, schema.Required,
		"evidence_id",
		"command",
		"cwd",
		"branch",
		"commit",
		"exit_code",
		"timestamp",
		"stdout_sha256",
		"artifact_path",
	)
	confidence := sortedStrings(schema.Properties["confidence"].Enum...)
	expectedConfidence := sortedStrings("high", "medium", "low")
	if !reflect.DeepEqual(confidence, expectedConfidence) {
		t.Fatalf("confidence enum drift:\nactual:   %#v\nexpected: %#v", confidence, expectedConfidence)
	}
	if got := schema.Properties["exit_code"].Type; got != "integer" {
		t.Fatalf("exit_code type = %q, want integer", got)
	}
}

func TestDownstreamAdoptionProofContractRequiredFields(t *testing.T) {
	schema := readSchema(t, "downstream-adoption-proof.schema.json")
	requireFields(t, schema.Required,
		"schema_version",
		"source_repo",
		"source_commit",
		"downstream_repo",
		"downstream_commit",
		"mode",
		"gate_outputs",
		"rollback",
	)
	mode := sortedStrings(schema.Properties["mode"].Enum...)
	expectedMode := sortedStrings("patch-only", "dry-run", "pr-plan")
	if !reflect.DeepEqual(mode, expectedMode) {
		t.Fatalf("mode enum drift:\nactual:   %#v\nexpected: %#v", mode, expectedMode)
	}
}

func TestExecutionEvidenceContractMatchesEvidenceManifest(t *testing.T) {
	manifest, err := os.ReadFile("../.agent/evidence/evidence-artifacts.yaml")
	if err != nil {
		t.Fatalf("read evidence-artifacts.yaml: %v", err)
	}
	text := string(manifest)
	for _, needle := range []string{
		"contracts/execution-evidence.schema.json",
		"execution_evidence",
		"evidence_id",
		"stdout_sha256",
		"exit_code",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf(".agent/evidence/evidence-artifacts.yaml missing required marker %q", needle)
		}
	}
}

func requireSchemaFieldMapsToStructField(t *testing.T, schema objectSchema, structType reflect.Type, schemaField string, structField string, schemaType string) {
	t.Helper()

	property, ok := schema.Properties[schemaField]
	if !ok {
		t.Fatalf("schema missing property %q", schemaField)
	}
	if property.Type != schemaType {
		t.Fatalf("schema property %q type = %q, want %q", schemaField, property.Type, schemaType)
	}
	if _, ok := structType.FieldByName(structField); !ok {
		t.Fatalf("%s missing field %s required by schema property %q", structType.Name(), structField, schemaField)
	}
}

func readText(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func readSchema(t *testing.T, path string) objectSchema {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var schema objectSchema
	if err := json.Unmarshal(content, &schema); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return schema
}

func requireFields(t *testing.T, actual []string, expected ...string) {
	t.Helper()
	fields := make(map[string]bool, len(actual))
	for _, field := range actual {
		fields[field] = true
	}
	for _, field := range expected {
		if !fields[field] {
			t.Fatalf("required fields missing %q from %#v", field, actual)
		}
	}
}

func sortedStrings(values ...string) []string {
	copied := append([]string(nil), values...)
	sort.Strings(copied)
	return copied
}

func redisxErrorIdentifiers() []redisx.ErrorIdentifier {
	return []redisx.ErrorIdentifier{
		redisx.ErrNil,
		redisx.ErrTimeout,
		redisx.ErrCanceled,
		redisx.ErrNetwork,
		redisx.ErrAuth,
		redisx.ErrReadOnly,
		redisx.ErrLoading,
		redisx.ErrTryAgain,
		redisx.ErrClusterMoved,
		redisx.ErrClusterAsk,
		redisx.ErrConnectionClosed,
		redisx.ErrInvalidConfig,
		redisx.ErrProvider,
	}
}

func redisxMetricNames() []string {
	return []string{
		redisx.MetricClientCreatedTotal,
		redisx.MetricClientClosedTotal,
		redisx.MetricClientErrorsTotal,
		redisx.MetricClientHealthStatus,
		redisx.MetricClientHealthLatencyMS,
		redisx.MetricClientRequestsTotal,
		redisx.MetricClientRequestDurationSeconds,
		redisx.MetricClientRetriesTotal,
		redisx.MetricClientInflight,
		redisx.MetricRedisOperationsTotal,
		redisx.MetricRedisOperationDurationSeconds,
		redisx.MetricRedisErrorsTotal,
		redisx.MetricRedisPoolConnections,
		redisx.MetricRedisHealthStatus,
	}
}

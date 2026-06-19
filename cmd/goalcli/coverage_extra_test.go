package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ZoneCNH/redisx/internal/debtcheck"
)

func TestSchemaHelperCoverageBranches(t *testing.T) {
	var single schemaType
	if err := json.Unmarshal([]byte(`"object"`), &single); err != nil {
		t.Fatalf("unmarshal single schema type: %v", err)
	}
	if len(single) != 1 || single[0] != "object" {
		t.Fatalf("single schema type = %#v; want object", single)
	}
	var many schemaType
	if err := json.Unmarshal([]byte(`["object","null"]`), &many); err != nil {
		t.Fatalf("unmarshal many schema types: %v", err)
	}
	var bad schemaType
	if err := json.Unmarshal([]byte(`123`), &bad); err == nil {
		t.Fatal("schemaType accepted numeric JSON")
	}

	values := map[string]struct {
		value any
		typ   string
		want  bool
	}{
		"object":   {value: map[string]any{}, typ: "object", want: true},
		"array":    {value: []any{}, typ: "array", want: true},
		"string":   {value: "x", typ: "string", want: true},
		"integer":  {value: float64(2), typ: "integer", want: true},
		"number":   {value: float64(2.5), typ: "number", want: true},
		"boolean":  {value: true, typ: "boolean", want: true},
		"null":     {value: nil, typ: "null", want: true},
		"unknown":  {value: "x", typ: "custom", want: true},
		"mismatch": {value: "x", typ: "object", want: false},
	}
	for name, tc := range values {
		if got := valueMatchesType(tc.value, tc.typ); got != tc.want {
			t.Fatalf("%s valueMatchesType = %t; want %t", name, got, tc.want)
		}
	}
	if !schemaAllowsType(jsonSchema{}, "object") {
		t.Fatal("schemaAllowsType rejected schema with no explicit type")
	}
	if schemaAllowsType(jsonSchema{Type: schemaType{"string"}}, "object") {
		t.Fatal("schemaAllowsType accepted object for string-only schema")
	}

	minLength := 3
	minItems := 2
	minimum := 10.0
	schema := jsonSchema{
		Type:     schemaType{"object"},
		Required: []string{"missing"},
		Properties: map[string]jsonSchema{
			"name":  {Type: schemaType{"string"}, MinLength: &minLength, Pattern: "["},
			"code":  {Type: schemaType{"string"}, Pattern: "^ok$"},
			"items": {Type: schemaType{"array"}, MinItems: &minItems, Items: &jsonSchema{Type: schemaType{"integer"}}},
			"age":   {Type: schemaType{"number"}, Minimum: &minimum},
			"kind":  {Type: schemaType{"string"}, Const: "required", Enum: []any{"allowed"}},
		},
	}
	value := map[string]any{
		"name":  "xy",
		"code":  "bad",
		"items": []any{float64(1.5)},
		"age":   float64(9),
		"kind":  "other",
	}
	gaps := validateValueAgainstSchema(value, schema, "artifact")
	for _, want := range []string{
		`artifact missing required field "missing"`,
		"artifact.name expected minLength 3",
		"artifact.name invalid schema pattern",
		"artifact.code value does not match pattern",
		"artifact.items expected at least 2 item(s)",
		"artifact.items[0] expected type integer",
		"artifact.age expected minimum 10",
		"artifact.kind expected const required",
		"artifact.kind expected enum",
	} {
		if !coverageSliceContains(gaps, want) {
			t.Fatalf("gaps = %#v; want %q", gaps, want)
		}
	}
	if gaps := validateValueAgainstSchema("not-object", jsonSchema{Type: schemaType{"object"}}, "artifact"); !coverageSliceContains(gaps, "artifact expected type object") {
		t.Fatalf("type mismatch gaps = %#v; want object type gap", gaps)
	}
}

func TestSchemaFixtureAndYAMLHelperBranches(t *testing.T) {
	schemas := map[string]jsonSchema{
		"zeta":  {Title: "Z"},
		"alpha": {Title: "A"},
		"event": {Title: "E"},
	}
	path, schema := selectFixtureSchema(filepath.Join("valid", "event.json"), schemas)
	if path != filepath.ToSlash(filepath.Join("schemas", "event.schema.json")) || schema.Title != "E" {
		t.Fatalf("selectFixtureSchema known = %s %#v; want event schema", path, schema)
	}
	path, schema = selectFixtureSchema(filepath.Join("valid", "unknown.json"), schemas)
	if path != filepath.ToSlash(filepath.Join("schemas", "alpha.schema.json")) || schema.Title != "A" {
		t.Fatalf("selectFixtureSchema fallback = %s %#v; want first sorted schema", path, schema)
	}
	if got := schemaFixtureKey(filepath.Join("schemas", "thing.schema.json")); got != "thing" {
		t.Fatalf("schemaFixtureKey = %q; want thing", got)
	}

	parsed, err := parseBaselineYAML(`name: "redisx # not comment"
count: 2
enabled: true
items:
  - id: a
    tags: [one, two]
`, "fixture.yaml")
	if err != nil {
		t.Fatalf("parseBaselineYAML returned error: %v", err)
	}
	if parsed["name"] != "redisx # not comment" || parsed["count"] != float64(2) || parsed["enabled"] != true {
		t.Fatalf("parsed scalars = %#v; want string, number, bool", parsed)
	}
	items, ok := parsed["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %#v; want one list item", parsed["items"])
	}
	for _, input := range []string{
		"bad",
		"items:\n  - bad\n",
		"items:\n  - id: a\n    bad\n",
	} {
		if _, err := parseBaselineYAML(input, "fixture.yaml"); err == nil {
			t.Fatalf("parseBaselineYAML(%q) returned nil error", input)
		}
	}
	if _, err := parseBaselineYAMLFile(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("parseBaselineYAMLFile returned nil error for missing file")
	}
	if err := writeSchemaCheckReport("", []byte("ignored")); err != nil {
		t.Fatalf("writeSchemaCheckReport empty path returned error: %v", err)
	}
	reportPath := filepath.Join(t.TempDir(), "nested", "report.json")
	if err := writeSchemaCheckReport(reportPath, []byte("{}")); err != nil {
		t.Fatalf("writeSchemaCheckReport returned error: %v", err)
	}
	if _, err := os.Stat(reportPath); err != nil {
		t.Fatalf("report file missing: %v", err)
	}
}

func TestVerifyArtifactExistsBranches(t *testing.T) {
	root := t.TempDir()
	coverageChdir(t, root)
	if err := os.MkdirAll(filepath.Join(root, "docs", "nested"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "file.md"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("write docs file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "notdir"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("write notdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "release", "debt"), 0o755); err != nil {
		t.Fatalf("mkdir release debt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "release", "debt", "latest.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write debt report: %v", err)
	}
	for _, artifact := range []string{"docs/file.md", "docs/*", "release/debt/latest.json"} {
		if err := verifyArtifactExists(artifact); err != nil {
			t.Fatalf("verifyArtifactExists(%q) returned error: %v", artifact, err)
		}
	}
	for _, artifact := range []string{"missing.md", "notdir/*", "missing/*", "[/*"} {
		if err := verifyArtifactExists(artifact); err == nil {
			t.Fatalf("verifyArtifactExists(%q) returned nil error", artifact)
		}
	}
}

func TestDebtHelperCoverageBranches(t *testing.T) {
	root := t.TempDir()
	coverageChdir(t, root)
	report := debtcheck.Report{
		SchemaVersion: debtcheck.SchemaVersion,
		Status:        "passed",
		Mode:          "enforce",
		Score:         8.5,
		MinScore:      debtcheck.DefaultMinScore,
		Sections: []debtcheck.SectionReport{
			{
				Name: "dependency",
				Findings: []debtcheck.Finding{
					{ID: "dep", Severity: "P1", Path: "go.mod", Message: "dependency drift"},
				},
			},
		},
	}
	if details := debtTrendDetails(report); !coverageSliceContains(details, "no prior debt evidence found") {
		t.Fatalf("details = %#v; want missing prior debt evidence", details)
	}
	if err := os.MkdirAll(filepath.Join(root, "release", "debt"), 0o755); err != nil {
		t.Fatalf("mkdir debt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "release", "debt", "latest.json"), []byte("{bad"), 0o644); err != nil {
		t.Fatalf("write bad latest: %v", err)
	}
	if details := debtTrendDetails(report); !coverageSliceContains(details, "is not a debt report") {
		t.Fatalf("details = %#v; want invalid prior debt report", details)
	}
	previous := debtcheck.Report{SchemaVersion: debtcheck.SchemaVersion, Status: "failed", Score: 7}
	previousData, err := json.Marshal(previous)
	if err != nil {
		t.Fatalf("marshal previous: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "release", "debt", "latest.json"), previousData, 0o644); err != nil {
		t.Fatalf("write previous latest: %v", err)
	}
	if details := debtTrendDetails(report); !coverageSliceContains(details, "score delta 1.50") {
		t.Fatalf("details = %#v; want score delta", details)
	}

	if suggestions := debtPatchSuggestions(debtcheck.Report{}); !coverageSliceContains(suggestions, "no patch suggestions") {
		t.Fatalf("suggestions = %#v; want no-findings suggestion", suggestions)
	}
	var findings []debtcheck.Finding
	for i := 0; i < 25; i++ {
		findings = append(findings, debtcheck.Finding{ID: "id", Severity: "P2", Path: "p", Message: "m"})
	}
	suggestions := debtPatchSuggestions(debtcheck.Report{Sections: []debtcheck.SectionReport{{Name: "section", Findings: findings}}})
	if len(suggestions) != 20 {
		t.Fatalf("suggestions len = %d; want cap at 20", len(suggestions))
	}

	var stdout, stderr bytes.Buffer
	if code := runDebtEvidence([]string{"extra"}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "does not accept arguments") {
		t.Fatalf("runDebtEvidence args code=%d stderr=%q; want usage error", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runDebtHelper("trend", []string{"unexpected"}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "does not accept positional argument") {
		t.Fatalf("runDebtHelper unexpected code=%d stderr=%q; want usage error", code, stderr.String())
	}
}

func TestRunDebtEvidenceWritesArtifacts(t *testing.T) {
	root := t.TempDir()
	coverageChdir(t, root)
	writeDebtPolicyInputsForCoverage(t, root)
	writeFileForCoverage(t, root, "safe.go", "package fixture\n")

	var stdout, stderr bytes.Buffer
	code := runDebtEvidence(nil, &stdout, &stderr)
	if code == 2 {
		t.Fatalf("runDebtEvidence code=%d stderr=%q; want debtcheck to run", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "wrote release/debt/latest.json") {
		t.Fatalf("stdout = %q; want written artifact message", stdout.String())
	}
	for _, path := range []string{
		filepath.Join(root, "release", "debt", "latest.json"),
		filepath.Join(root, "release", "debt", "latest.md"),
		filepath.Join(root, "release", "debt", "latest.json.sha256"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected debt artifact %s: %v", path, err)
		}
	}
}

func TestGovernanceMakefileHelperBranches(t *testing.T) {
	makefile := strings.Join([]string{
		"context-lite: require-gowork-off governance-check forbidden",
		"\t@echo lite",
		"context-standard: context-lite \\",
		" docs-check",
		"\t@echo standard",
		"context-release: context-full release-final-check mystery",
		"\t$(MAKE) release-final-check",
		"release-final-check:",
		"\t$(MAKE) release-final-check",
		"",
	}, "\n")

	if block := makefileTargetBlock(makefile, "context-standard"); !strings.Contains(block, "docs-check") {
		t.Fatalf("context-standard block = %q; want continuation dependency", block)
	}
	if got := makefileTargetDefinitionCount(makefile+"\ncontext-lite:\n", "context-lite"); got != 2 {
		t.Fatalf("makefileTargetDefinitionCount = %d; want duplicate count 2", got)
	}
	targets := makefileTargetNames(".PHONY: context-lite\nVAR := value\ncontext-lite:\n\t@echo ok\n")
	if !targets["context-lite"] || targets["VAR"] || targets[".PHONY"] {
		t.Fatalf("makefileTargetNames = %#v; want only context-lite target", targets)
	}
	deps := makefileTargetDependencies(makefile, "context-standard")
	if !makefileDependencyHasToken(deps, "context-lite") || !makefileDependencyHasToken(deps, "docs-check") {
		t.Fatalf("context-standard deps = %#v; want context-lite and docs-check", deps)
	}
	if deps := makefileTargetDependencies("broken-target\n", "broken-target"); deps != nil {
		t.Fatalf("broken deps = %#v; want nil", deps)
	}

	var gaps []string
	appendMakefileDuplicateGaps(makefile, []string{"missing-target", "context-lite"}, &gaps)
	appendMakefileTargetDependencyGaps(makefile, "missing-target", []string{"required"}, nil, &gaps)
	appendMakefileTargetDependencyGaps(makefile, "context-lite", []string{"required"}, []string{"forbidden"}, &gaps)
	appendMakefileTargetForbiddenReferenceGaps(makefile, "missing-target", []string{"forbidden"}, &gaps)
	appendMakefileTargetForbiddenReferenceGaps(makefile, "context-release", []string{"release-final-check"}, &gaps)
	appendReleaseFinalDelegationGaps("release-final-check:\n\t@echo no delegation\n", &gaps)
	appendContextProfileDAGGaps(makefile, &gaps)
	for _, want := range []string{
		"missing-target must be defined exactly once",
		"missing target block missing-target",
		"context-lite missing dependency required",
		"context-lite must not depend on forbidden",
		"context-release must not reference release-final-check",
		"release-final-check must call context-release",
		"context-release references unknown context gate mystery",
		"context-release must not reach release-final-check",
	} {
		if !coverageSliceContains(gaps, want) {
			t.Fatalf("gaps = %#v; want %q", gaps, want)
		}
	}

	var cycleGaps []string
	appendMakefileProfileCycleGaps(map[string][]string{
		"context-lite":     {"context-standard"},
		"context-standard": {"context-lite"},
	}, []string{"context-lite"}, &cycleGaps)
	if !coverageSliceContains(cycleGaps, "Makefile context profile DAG cycle") {
		t.Fatalf("cycleGaps = %#v; want profile cycle", cycleGaps)
	}
	if !makefileGraphReaches(map[string][]string{"a": {"b"}, "b": {"c"}}, "a", "c", map[string]bool{}) {
		t.Fatal("makefileGraphReaches did not find reachable target")
	}
	if makefileGraphReaches(map[string][]string{"a": {"b"}, "b": {"a"}}, "a", "c", map[string]bool{}) {
		t.Fatal("makefileGraphReaches reported unreachable cycle target")
	}
}

func TestGovernanceArgumentAndScalarHelpers(t *testing.T) {
	if normalizeContextProfile("fast") != "lite" || normalizeContextProfile("full") != "full" {
		t.Fatal("normalizeContextProfile returned unexpected values")
	}
	for command, want := range map[string]string{
		"context-lite":       "lite",
		"context-fast-check": "lite",
		"context-full":       "full",
		"context-full-check": "full",
		"context-release":    "release",
		"other":              "standard",
	} {
		if got := mapContextAliasToProfile(command); got != want {
			t.Fatalf("mapContextAliasToProfile(%q) = %q; want %q", command, got, want)
		}
	}
	if !validContextProfileName("lite") || validContextProfileName("fast") {
		t.Fatal("validContextProfileName returned unexpected values")
	}
	if profile, ok := contextGateProfile("context-standard-check"); !ok || profile != "standard" {
		t.Fatalf("contextGateProfile standard = %q %t; want standard true", profile, ok)
	}
	if _, ok := contextGateProfile("unknown"); ok {
		t.Fatal("contextGateProfile accepted unknown gate")
	}
	if got := stripInlineComment("value # comment"); got != "value " {
		t.Fatalf("stripInlineComment = %q; want prefix before comment", got)
	}
	if got := fallback("", "fallback"); got != "fallback" {
		t.Fatalf("fallback empty = %q; want fallback", got)
	}
	t.Setenv("REDISX_COVERAGE_ENV", "set")
	if got := envDefault("REDISX_COVERAGE_ENV", "fallback"); got != "set" {
		t.Fatalf("envDefault set = %q; want set", got)
	}
	if got := envDefault("REDISX_COVERAGE_MISSING", "fallback"); got != "fallback" {
		t.Fatalf("envDefault missing = %q; want fallback", got)
	}
	if !validContext("local_write") || validContext("invalid") {
		t.Fatal("validContext returned unexpected values")
	}
	if !flagProvided([]string{"--json", "--repo=."}, "repo") || flagProvided([]string{"--profile"}, "repo") {
		t.Fatal("flagProvided returned unexpected values")
	}
	if err := validatePlannedCommandArgs("planned", []string{"--context", "invalid"}); err == nil || !strings.Contains(err.Error(), "invalid context") {
		t.Fatalf("validatePlannedCommandArgs invalid context err = %v; want invalid context", err)
	}
	if err := validatePlannedCommandArgs("planned", []string{"positional"}); err == nil || !strings.Contains(err.Error(), "unexpected positional argument") {
		t.Fatalf("validatePlannedCommandArgs positional err = %v; want positional error", err)
	}
	if err := validateInternalCommandArgs("internal", []string{"--dry-run", "--output=out"}, internalCommandFlagSpec{boolFlags: []string{"dry-run"}, stringFlags: []string{"output"}}); err != nil {
		t.Fatalf("validateInternalCommandArgs valid args returned error: %v", err)
	}
	if err := validateInternalCommandArgs("internal", []string{"positional"}, internalCommandFlagSpec{}); err == nil || !strings.Contains(err.Error(), "unexpected positional argument") {
		t.Fatalf("validateInternalCommandArgs positional err = %v; want positional error", err)
	}
	var stderr bytes.Buffer
	if code := invalidInternalArgsExit("internal", flag.ErrHelp, &stderr); code != 0 || stderr.Len() != 0 {
		t.Fatalf("invalidInternalArgsExit help code=%d stderr=%q; want silent success", code, stderr.String())
	}
	if code := invalidInternalArgsExit("internal", os.ErrInvalid, &stderr); code != 2 || !strings.Contains(stderr.String(), "invalid arguments") {
		t.Fatalf("invalidInternalArgsExit error code=%d stderr=%q; want usage error", code, stderr.String())
	}
}

func coverageChdir(t *testing.T, dir string) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
}

func coverageSliceContains(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

func writeDebtPolicyInputsForCoverage(t *testing.T, root string) {
	t.Helper()
	for _, path := range []string{
		debtcheck.DefaultRulesPath,
		debtcheck.DefaultRegistryPath,
		debtcheck.DefaultExceptions,
		debtcheck.DefaultPurpose,
	} {
		writeFileForCoverage(t, root, path, "schema_version: debt/v1\n")
	}
}

func writeFileForCoverage(t *testing.T, root, path, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

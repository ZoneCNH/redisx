package scripts_test

import (
	"os"
	"strings"
	"testing"
)

func TestRunIntegrationGeneratesStandardImpactBeforeReleaseEvidence(t *testing.T) {
	contents, err := os.ReadFile("run_integration.sh")
	if err != nil {
		t.Fatalf("read run_integration.sh: %v", err)
	}

	script := string(contents)
	standardImpactIndex := strings.Index(script, "GOWORK=off make standard-impact-check")
	if standardImpactIndex < 0 {
		t.Fatal("run_integration.sh does not generate standard impact evidence")
	}
	debtIndex := strings.Index(script, "\n    GOWORK=off make debt\n")
	if debtIndex < 0 {
		t.Fatal("run_integration.sh does not run downstream debt gate")
	}
	debtEvidenceIndex := strings.Index(script, "GOWORK=off make debt-evidence")
	if debtEvidenceIndex < 0 {
		t.Fatal("run_integration.sh does not generate downstream debt evidence")
	}
	debtChecksumIndex := strings.Index(script, "GOWORK=off make debt-evidence-checksum-check")
	if debtChecksumIndex < 0 {
		t.Fatal("run_integration.sh does not verify downstream debt evidence checksum")
	}
	evidenceIndex := strings.Index(script, "retry_command 3 env CHECK_STATUS=passed GOWORK=off make evidence")
	if evidenceIndex < 0 {
		t.Fatal("run_integration.sh does not generate release evidence")
	}
	checkIndex := strings.Index(script, "retry_command 3 env RELEASE_EVIDENCE_REQUIRE_PASSED=1 GOWORK=off make release-evidence-check")
	if checkIndex < 0 {
		t.Fatal("run_integration.sh does not verify release evidence")
	}

	if standardImpactIndex > debtIndex || debtIndex > debtEvidenceIndex || debtEvidenceIndex > debtChecksumIndex || debtChecksumIndex > evidenceIndex || evidenceIndex > checkIndex {
		t.Fatalf(
			"integration evidence order is wrong: standard-impact=%d debt=%d debt-evidence=%d debt-checksum=%d evidence=%d release-check=%d",
			standardImpactIndex,
			debtIndex,
			debtEvidenceIndex,
			debtChecksumIndex,
			evidenceIndex,
			checkIndex,
		)
	}
}

func TestRunIntegrationRetriesNetworkSensitiveDownstreamCommands(t *testing.T) {
	contents, err := os.ReadFile("run_integration.sh")
	if err != nil {
		t.Fatalf("read run_integration.sh: %v", err)
	}

	script := string(contents)
	for _, target := range []string{
		"retry_command()",
		"retry_command 3 env GOWORK=off go mod tidy",
		"retry_command 3 env GOWORK=off go mod download all",
		"retry_command 3 env GOWORK=off go test ./...",
		"retry_command 3 env CHECK_STATUS=passed GOWORK=off make evidence",
		"retry_command 3 env RELEASE_EVIDENCE_REQUIRE_PASSED=1 GOWORK=off make release-evidence-check",
	} {
		if !strings.Contains(script, target) {
			t.Fatalf("run_integration.sh missing retry target %q", target)
		}
	}

	tidyIndex := strings.Index(script, "retry_command 3 env GOWORK=off go mod tidy")
	downloadIndex := strings.Index(script, "retry_command 3 env GOWORK=off go mod download all")
	diffIndex := strings.Index(script, "git diff --exit-code -- go.mod go.sum")
	if tidyIndex < 0 || downloadIndex < 0 || diffIndex < 0 {
		t.Fatalf("missing downstream module preparation commands: tidy=%d download=%d diff=%d", tidyIndex, downloadIndex, diffIndex)
	}
	if tidyIndex > downloadIndex || downloadIndex > diffIndex {
		t.Fatalf("go mod download must happen after tidy and before clean diff check: tidy=%d download=%d diff=%d", tidyIndex, downloadIndex, diffIndex)
	}
}

func TestRunIntegrationCoversRequiredDownstreams(t *testing.T) {
	contents, err := os.ReadFile("run_integration.sh")
	if err != nil {
		t.Fatalf("read run_integration.sh: %v", err)
	}

	script := string(contents)
	for _, target := range []string{
		"kernel|github.com/ZoneCNH/kernel|kernel",
		"configx|github.com/ZoneCNH/configx|configx",
		`standard_adapter_name="redis""x"`,
		`"${standard_adapter_name}|github.com/ZoneCNH/${standard_adapter_name}|${standard_adapter_name}"`,
	} {
		if !strings.Contains(script, target) {
			t.Fatalf("run_integration.sh missing downstream target %q", target)
		}
	}

	if strings.Contains(script, "corekit|example.com/acme/corekit|corekit") {
		t.Fatal("run_integration.sh still includes legacy corekit integration target")
	}
}

func TestRunIntegrationRunsRedisIntegrationProfile(t *testing.T) {
	integrationContents, err := os.ReadFile("run_integration.sh")
	if err != nil {
		t.Fatalf("read run_integration.sh: %v", err)
	}
	if !strings.Contains(string(integrationContents), "GOWORK=off make test-integration") {
		t.Fatal("run_integration.sh does not run the Redis integration profile")
	}

	makefileContents, err := os.ReadFile("../Makefile")
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	if !strings.Contains(string(makefileContents), "test-integration:\n\t./scripts/run_redis_integration.sh") {
		t.Fatal("Makefile test-integration target does not invoke the Redis integration runner")
	}

	redisIntegrationContents, err := os.ReadFile("run_redis_integration.sh")
	if err != nil {
		t.Fatalf("read run_redis_integration.sh: %v", err)
	}
	runner := string(redisIntegrationContents)
	for _, token := range []string{
		"REDISX_INTEGRATION=1",
		"REDISX_REDIS_ADDR",
		"REDISX_INTEGRATION_DOCKER",
		"docker restart",
		"redisx:persistence:marker",
	} {
		if !strings.Contains(runner, token) {
			t.Fatalf("Redis integration runner missing %q", token)
		}
	}
}

func TestRunIntegrationNormalizesDockerRenderedOwnershipBeforeGit(t *testing.T) {
	contents, err := os.ReadFile("run_integration.sh")
	if err != nil {
		t.Fatalf("read run_integration.sh: %v", err)
	}

	script := string(contents)
	checkIndex := strings.Index(script, `./scripts/check_rendered_template.sh "$out_dir" "$module_name" "$module_path" "$package_name"`)
	if checkIndex < 0 {
		t.Fatal("run_integration.sh does not check rendered templates")
	}
	ownershipIndex := strings.Index(script, `stat -c %u "$out_dir"`)
	if ownershipIndex < 0 {
		t.Fatal("run_integration.sh does not inspect rendered output ownership")
	}
	chownIndex := strings.Index(script, `chown -R "$(id -u):$(id -g)" "$out_dir"`)
	if chownIndex < 0 {
		t.Fatal("run_integration.sh does not normalize rendered output ownership")
	}
	gitIndex := strings.Index(script, "git init -q")
	if gitIndex < 0 {
		t.Fatal("run_integration.sh does not initialize rendered git repos")
	}

	if checkIndex > ownershipIndex || ownershipIndex > chownIndex || chownIndex > gitIndex {
		t.Fatalf(
			"rendered ownership normalization must happen after template checks and before git init: check=%d stat=%d chown=%d git=%d",
			checkIndex,
			ownershipIndex,
			chownIndex,
			gitIndex,
		)
	}
}

func TestRenderedTemplateCheckSkipsMigratedInboxArchive(t *testing.T) {
	contents, err := os.ReadFile("check_rendered_template.sh")
	if err != nil {
		t.Fatalf("read check_rendered_template.sh: %v", err)
	}

	script := string(contents)
	for _, token := range []string{
		"--glob '!.agent/archive/inbox/**'",
		"--glob '!**/.agent/archive/inbox/**'",
		"-not -path '*/.agent/archive/inbox/*'",
	} {
		if !strings.Contains(script, token) {
			t.Fatalf("check_rendered_template.sh missing migrated inbox archive exclusion %q", token)
		}
	}
}

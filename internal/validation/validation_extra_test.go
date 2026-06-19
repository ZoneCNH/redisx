package validation

import (
	"strings"
	"testing"
)

func TestValidateRuntimeFileOwnershipReportsMalformedAndMissingControls(t *testing.T) {
	if gaps := ValidateRuntimeFileOwnership("runtime.yaml", ""); !validationExtraGapsContain(gaps, "runtime.yaml must not be empty") {
		t.Fatalf("gaps = %#v; want empty manifest gap", gaps)
	}

	fixture := `owners:
  "/abs":
    owner: nobody
    review_required: maybe
  ".agent/":
    owner: gate-runtime
    review_required: true
    rationale: Control plane.
`
	gaps := ValidateRuntimeFileOwnership("runtime.yaml", fixture)
	for _, want := range []string{
		"runtime.yaml missing schema_version",
		"runtime.yaml /abs unknown owner nobody",
		"runtime.yaml /abs must be repository-relative",
		"runtime.yaml /abs review_required must be true or false",
		"runtime.yaml /abs missing rationale",
		"runtime.yaml .agent/ missing review_rule",
		"runtime.yaml .agent/ owner must be governance",
		"runtime.yaml owners must include cmd/goalcli/",
		"runtime.yaml owners must include contracts/",
	} {
		if !validationExtraGapsContain(gaps, want) {
			t.Fatalf("gaps = %#v; want %q", gaps, want)
		}
	}
}

func TestValidateExecutionContextReportsMalformedAndSemanticGaps(t *testing.T) {
	if gaps := ValidateExecutionContext("contexts.yaml", "", []string{"local_write"}); !validationExtraGapsContain(gaps, "contexts.yaml must not be empty") {
		t.Fatalf("gaps = %#v; want empty manifest gap", gaps)
	}

	fixture := `contexts:
  local_write:
    mutates_files: maybe
    release_evidence: true
    manifest_path: /tmp/manifest.yaml
  local_write:
    write_scope: duplicate
    mutates_files: true
    release_evidence: false
    requires_gowork: off
  unexpected:
    write_scope: repo
    mutates_files: false
    release_evidence: false
    requires_gowork: off
  release_verify:
    write_scope: verify
    mutates_files: true
    release_evidence: false
    requires_gowork: on
`
	gaps := ValidateExecutionContext("contexts.yaml", fixture, []string{"local_write", "release_verify", "release_final"})
	for _, want := range []string{
		"contexts.yaml missing schema_version",
		"contexts.yaml duplicate context local_write",
		"contexts.yaml unknown context unexpected",
		"contexts.yaml local_write missing write_scope",
		"contexts.yaml local_write mutates_files must be true or false",
		"contexts.yaml local_write missing requires_gowork",
		"contexts.yaml local_write manifest_path must be repository-relative",
		"contexts.yaml local_write release_evidence must be false",
		"contexts.yaml release_verify mutates_files must be false",
		"contexts.yaml release_verify release_evidence must be true",
		"contexts.yaml release_verify requires_gowork must be off",
		"contexts.yaml missing context release_final",
	} {
		if !validationExtraGapsContain(gaps, want) {
			t.Fatalf("gaps = %#v; want %q", gaps, want)
		}
	}
}

func validationExtraGapsContain(gaps []string, want string) bool {
	for _, gap := range gaps {
		if strings.Contains(gap, want) {
			return true
		}
	}
	return false
}

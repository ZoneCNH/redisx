package scripts_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckRenderedTemplateAllowsGoCompositeLiterals(t *testing.T) {
	repoDir := writeRenderedTemplateFixture(t, "kernel", "github.com/ZoneCNH/kernel", "kernel")
	writeRenderedFile(t, repoDir, "internal/provider/memory.go", `package provider

func keepCompositeLiteral() {
	result.Values = []Value{{Value: item.value, Found: true}}
}
`)

	cmd := exec.Command("bash", "check_rendered_template.sh", repoDir, "kernel", "github.com/ZoneCNH/kernel", "kernel")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("check_rendered_template.sh rejected valid Go composite literal: %v\n%s", err, output)
	}
}

func TestCheckRenderedTemplateRejectsTemplateIdentifiers(t *testing.T) {
	repoDir := writeRenderedTemplateFixture(t, "kernel", "github.com/ZoneCNH/kernel", "kernel")
	writeRenderedFile(t, repoDir, "README.md", "# {"+"{ .ModuleName }"+"}\n")

	cmd := exec.Command("bash", "check_rendered_template.sh", repoDir, "kernel", "github.com/ZoneCNH/kernel", "kernel")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("check_rendered_template.sh accepted unresolved template identifier:\n%s", output)
	}
	if !strings.Contains(string(output), "found stale template placeholder") {
		t.Fatalf("check_rendered_template.sh failed for the wrong reason:\n%s", output)
	}
}

func writeRenderedTemplateFixture(t *testing.T, moduleName, modulePath, packageName string) string {
	t.Helper()

	repoDir := t.TempDir()
	writeRenderedFile(t, repoDir, "go.mod", "module "+modulePath+"\n\ngo 1.25\n")
	writeRenderedFile(t, repoDir, filepath.Join("pkg", packageName, "doc.go"), "package "+packageName+"\n")

	for _, requiredPath := range []string{
		"Dockerfile",
		"docker-compose.yml",
		".dockerignore",
		".devcontainer/devcontainer.json",
		"scripts/docker/check_toolchain.sh",
		"scripts/docker/docker_gate.sh",
	} {
		writeRenderedFile(t, repoDir, requiredPath, "\n")
	}

	targets := []string{
		"docker-toolchain-check",
		"docker-build",
		"docker-build-check",
		"docker-shell",
		"docker-ci",
		"docker-release-check",
		"docker-release-final-check",
		"docker-goalcli",
		"docker-goalcli-image",
		"docker-goalcli-version",
		"docker-runtime-check",
		"docker-drift-check",
		"docker-contract",
	}

	var makefile strings.Builder
	makefile.WriteString(".PHONY:")
	for _, target := range targets {
		makefile.WriteByte(' ')
		makefile.WriteString(target)
	}
	makefile.WriteString("\n")
	for _, target := range targets {
		makefile.WriteString(target)
		makefile.WriteString(":\n\t@true\n")
	}
	writeRenderedFile(t, repoDir, "Makefile", makefile.String())

	_ = moduleName
	return repoDir
}

func writeRenderedFile(t *testing.T, root, path, contents string) {
	t.Helper()

	fullPath := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("create parent for %s: %v", path, err)
	}
	if err := os.WriteFile(fullPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

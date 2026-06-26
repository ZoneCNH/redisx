package redisx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlan008READMEContractDocumentsTopologyDegradation(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	readme := string(data)

	for _, token := range []string{
		"sentinel",
		"cluster",
		"MOVED/ASK",
		"fallback",
		"degraded health",
	} {
		if !strings.Contains(readme, token) {
			t.Fatalf("README.md must document %q", token)
		}
	}
}

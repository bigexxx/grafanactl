package local

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grafana/grafanactl/internal/format"
	"github.com/grafana/grafanactl/internal/resources"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestFSReader_SkipsAlertsDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "Alerts"), 0o755))

	// This is intentionally not a Kubernetes-style object.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Alerts", "rule.json"), []byte(`{"uid":"abc"}`), 0o644))

	r := FSReader{
		Decoders: format.Codecs(),
	}

	dst := resources.NewResources()
	err := r.Read(t.Context(), dst, nil, []string{dir})
	require.NoError(t, err)
	require.Equal(t, 0, dst.Len())

	// Ensure we didn't accidentally decode something from Alerts.
	for _, res := range dst.AsList() {
		require.NotEqual(t, schema.GroupVersionKind{}, res.GroupVersionKind())
	}
}

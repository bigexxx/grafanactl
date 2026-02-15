package resources

import (
	"testing"

	internalresources "github.com/grafana/grafanactl/internal/resources"
)

func TestAppendSyntheticDescriptors_AppendsAlerts(t *testing.T) {
	in := internalresources.Descriptors{
		{Plural: "dashboards"},
	}
	out := appendSyntheticDescriptors(in)
	if len(out) != len(in)+1 {
		t.Fatalf("expected %d descriptors, got %d", len(in)+1, len(out))
	}
	if out[len(out)-1].Plural != "alerts" {
		t.Fatalf("expected last descriptor to be alerts, got %q", out[len(out)-1].Plural)
	}
}

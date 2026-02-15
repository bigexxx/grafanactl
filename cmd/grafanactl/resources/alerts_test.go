package resources

import "testing"

func TestSplitAlertsSelectors(t *testing.T) {
	t.Run("no args", func(t *testing.T) {
		alerts, uids, other := splitAlertsSelectors(nil)
		if alerts {
			t.Fatalf("expected alertsRequested=false")
		}
		if uids != nil {
			t.Fatalf("expected alertUIDs=nil")
		}
		if other != nil {
			t.Fatalf("expected otherSelectors=nil")
		}
	})

	t.Run("alerts only", func(t *testing.T) {
		alerts, uids, other := splitAlertsSelectors([]string{"alerts"})
		if !alerts {
			t.Fatalf("expected alertsRequested=true")
		}
		if len(uids) != 0 {
			t.Fatalf("expected no uids, got %v", uids)
		}
		if len(other) != 0 {
			t.Fatalf("expected no other selectors, got %v", other)
		}
	})

	t.Run("alerts with uids", func(t *testing.T) {
		alerts, uids, other := splitAlertsSelectors([]string{"alerts/a,b"})
		if !alerts {
			t.Fatalf("expected alertsRequested=true")
		}
		if len(uids) != 2 || uids[0] != "a" || uids[1] != "b" {
			t.Fatalf("unexpected uids: %v", uids)
		}
		if len(other) != 0 {
			t.Fatalf("expected no other selectors, got %v", other)
		}
	})

	t.Run("mixed selectors", func(t *testing.T) {
		alerts, uids, other := splitAlertsSelectors([]string{"dashboards", "alerts", "folders"})
		if !alerts {
			t.Fatalf("expected alertsRequested=true")
		}
		if len(uids) != 0 {
			t.Fatalf("expected no uids, got %v", uids)
		}
		if len(other) != 2 || other[0] != "dashboards" || other[1] != "folders" {
			t.Fatalf("unexpected other selectors: %v", other)
		}
	})
}

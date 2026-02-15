package discovery

import (
	"slices"
	"testing"
)

func TestIgnoredResourceGroups_DoesNotIgnoreAlertingNotifications(t *testing.T) {
	if slices.Contains(ignoredResourceGroups, "notifications.alerting.grafana.app") {
		t.Fatalf("notifications.alerting.grafana.app must not be ignored; alert resources should be discoverable")
	}
}

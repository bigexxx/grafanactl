package resources

import (
	internalresources "github.com/grafana/grafanactl/internal/resources"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var syntheticDescriptors = internalresources.Descriptors{
	{
		// Not a Kubernetes-style resource on most Grafana instances; we fetch it via the provisioning API.
		GroupVersion: schema.GroupVersion{Group: "provisioning.alerting.grafana", Version: "v1"},
		Kind:         "AlertRules",
		Singular:     "alerts",
		Plural:       "alerts",
	},
}

func appendSyntheticDescriptors(descs internalresources.Descriptors) internalresources.Descriptors {
	out := make(internalresources.Descriptors, 0, len(descs)+len(syntheticDescriptors))
	out = append(out, descs...)
	out = append(out, syntheticDescriptors...)
	return out
}

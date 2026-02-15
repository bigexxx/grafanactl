package resources

import (
	"context"

	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/resources"
	"github.com/grafana/grafanactl/internal/resources/remote"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func syncDeleteResources(
	ctx context.Context,
	cfg config.NamespacedRESTConfig,
	supported map[schema.GroupVersionKind]resources.Descriptor,
	filters resources.Filters,
	localResources *resources.Resources,
	maxConcurrent int,
	stopOnError bool,
	dryRun bool,
	includeManaged bool,
) (deleted int, failed int, err error) {
	descriptors := syncDescriptors(filters, localResources, supported)
	if len(descriptors) == 0 {
		return 0, 0, nil
	}

	localIdx := make(map[objKey]struct{}, localResources.Len())
	for _, r := range localResources.AsList() {
		localIdx[objKey{gvk: r.GroupVersionKind(), name: r.Name()}] = struct{}{}
	}

	puller, err := remote.NewDefaultPuller(ctx, cfg)
	if err != nil {
		return 0, 0, err
	}

	remoteRes := resources.NewResources()
	pullFilters := make(resources.Filters, 0, len(descriptors))
	for _, d := range descriptors {
		pullFilters = append(pullFilters, resources.Filter{
			Type:       resources.FilterTypeAll,
			Descriptor: d,
		})
	}

	if err := puller.Pull(ctx, remote.PullRequest{
		Filters:        pullFilters,
		Resources:      remoteRes,
		StopOnError:    stopOnError,
		ExcludeManaged: !includeManaged,
	}); err != nil {
		return 0, 0, err
	}

	toDelete := resources.NewResources()
	for _, r := range remoteRes.AsList() {
		if _, ok := localIdx[objKey{gvk: r.GroupVersionKind(), name: r.Name()}]; ok {
			continue
		}
		toDelete.Add(r)
	}

	if toDelete.Len() == 0 {
		return 0, 0, nil
	}

	deleter, err := remote.NewDeleter(ctx, cfg)
	if err != nil {
		return 0, 0, err
	}

	summary, err := deleter.Delete(ctx, remote.DeleteRequest{
		Resources:      toDelete,
		MaxConcurrency: maxConcurrent,
		StopOnError:    stopOnError,
		DryRun:         dryRun,
	})
	if err != nil {
		return summary.DeletedCount, summary.FailedCount, err
	}

	return summary.DeletedCount, summary.FailedCount, nil
}

func syncDescriptors(
	filters resources.Filters,
	localResources *resources.Resources,
	supported map[schema.GroupVersionKind]resources.Descriptor,
) []resources.Descriptor {
	if len(filters) > 0 {
		seen := make(map[schema.GroupVersionKind]struct{}, len(filters))
		out := make([]resources.Descriptor, 0, len(filters))
		for _, f := range filters {
			gvk := f.Descriptor.GroupVersionKind()
			if _, ok := seen[gvk]; ok {
				continue
			}
			seen[gvk] = struct{}{}
			out = append(out, f.Descriptor)
		}
		return out
	}

	seen := make(map[schema.GroupVersionKind]struct{})
	out := make([]resources.Descriptor, 0)
	for _, r := range localResources.AsList() {
		gvk := r.GroupVersionKind()
		if _, ok := seen[gvk]; ok {
			continue
		}
		seen[gvk] = struct{}{}

		desc, ok := supported[gvk]
		if !ok {
			// If we can't resolve a full descriptor, skip syncing this kind.
			continue
		}
		out = append(out, desc)
	}

	return out
}

type objKey struct {
	gvk  schema.GroupVersionKind
	name string
}

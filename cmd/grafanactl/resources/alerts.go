package resources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	openapiruntime "github.com/go-openapi/runtime"
	"github.com/grafana/grafana-openapi-client-go/client/provisioning"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/format"
	"github.com/grafana/grafanactl/internal/grafana"
)

const (
	alertsSelector = "alerts"
	alertsDirName  = "Alerts"
)

func pullAlerts(
	ctx context.Context,
	cfg *config.Context,
	outDir string,
	outputFormat string,
	codec format.Codec,
	alertUIDs []string,
) (int, error) {
	if outputFormat != "json" && outputFormat != "yaml" {
		return 0, fmt.Errorf("alerts pull only supports -o json or -o yaml (got %q)", outputFormat)
	}

	gClient, err := grafana.ClientFromContext(cfg)
	if err != nil {
		return 0, err
	}

	listResp, err := gClient.Provisioning.GetAlertRulesWithParams(provisioning.NewGetAlertRulesParams().WithContext(ctx))
	if err != nil {
		return 0, err
	}

	if err := os.MkdirAll(filepath.Join(outDir, alertsDirName), 0o755); err != nil {
		return 0, err
	}

	uidSet := make(map[string]struct{}, len(alertUIDs))
	for _, uid := range alertUIDs {
		uid = strings.TrimSpace(uid)
		if uid == "" {
			continue
		}
		uidSet[uid] = struct{}{}
	}

	pulled := 0
	for _, r := range listResp.Payload {
		uid := strings.TrimSpace(r.UID)
		if uid == "" {
			continue
		}

		if len(uidSet) > 0 {
			if _, ok := uidSet[uid]; !ok {
				continue
			}
		}

		getParams := provisioning.NewGetAlertRuleParams().WithUID(uid).WithContext(ctx)
		getResp, err := gClient.Provisioning.GetAlertRuleWithParams(getParams)
		if err != nil {
			return pulled, err
		}

		filename := filepath.Join(outDir, alertsDirName, uid+"."+outputFormat)
		f, err := os.Create(filename)
		if err != nil {
			return pulled, err
		}

		if err := codec.Encode(f, getResp.Payload); err != nil {
			_ = f.Close()
			return pulled, err
		}
		_ = f.Close()

		pulled++
	}

	return pulled, nil
}

func splitAlertsSelectors(args []string) (alertsRequested bool, alertUIDs []string, otherSelectors []string) {
	if len(args) == 0 {
		return false, nil, nil
	}

	other := make([]string, 0, len(args))
	for _, a := range args {
		a = strings.TrimSpace(a)
		switch {
		case a == alertsSelector || a == "alert-rules" || a == "alertrules":
			alertsRequested = true
		case strings.HasPrefix(a, alertsSelector+"/"):
			alertsRequested = true
			parts := strings.TrimPrefix(a, alertsSelector+"/")
			alertUIDs = appendCSVNonEmpty(alertUIDs, parts)
		case strings.HasPrefix(a, "alert-rules/"):
			alertsRequested = true
			parts := strings.TrimPrefix(a, "alert-rules/")
			alertUIDs = appendCSVNonEmpty(alertUIDs, parts)
		default:
			other = append(other, a)
		}
	}

	return alertsRequested, alertUIDs, other
}

func appendCSVNonEmpty(dst []string, csv string) []string {
	for _, s := range strings.Split(csv, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		dst = append(dst, s)
	}
	return dst
}

func pushAlerts(
	ctx context.Context,
	cfg *config.Context,
	paths []string,
	stopOnError bool,
	dryRun bool,
) (pushed int, failed int, err error) {
	rules, err := readAlertRuleFiles(paths)
	if err != nil {
		return 0, 0, err
	}
	if len(rules) == 0 {
		return 0, 0, nil
	}

	if dryRun {
		return len(rules), 0, nil
	}

	gClient, err := grafana.ClientFromContext(cfg)
	if err != nil {
		return 0, 0, err
	}

	for _, rule := range rules {
		uid := strings.TrimSpace(rule.UID)
		if uid == "" {
			failed++
			if stopOnError {
				return pushed, failed, fmt.Errorf("alert rule is missing uid")
			}
			continue
		}

		// Prefer update; if the rule doesn't exist, create it.
		putParams := provisioning.NewPutAlertRuleParams().WithUID(uid).WithBody(rule).WithContext(ctx)
		if _, err := gClient.Provisioning.PutAlertRule(putParams); err != nil {
			if apiErr, ok := err.(*openapiruntime.APIError); ok && apiErr.Code == 404 {
				postParams := provisioning.NewPostAlertRuleParams().WithBody(rule).WithContext(ctx)
				if _, postErr := gClient.Provisioning.PostAlertRule(postParams); postErr != nil {
					failed++
					if stopOnError {
						return pushed, failed, postErr
					}
					continue
				}
				pushed++
				continue
			}

			failed++
			if stopOnError {
				return pushed, failed, err
			}
			continue
		}

		pushed++
	}

	return pushed, failed, nil
}

func syncDeleteAlerts(
	ctx context.Context,
	cfg *config.Context,
	paths []string,
	stopOnError bool,
	dryRun bool,
) (deleted int, failed int, err error) {
	localRules, err := readAlertRuleFiles(paths)
	if err != nil {
		return 0, 0, err
	}

	localUIDs := make(map[string]struct{}, len(localRules))
	for _, r := range localRules {
		uid := strings.TrimSpace(r.UID)
		if uid == "" {
			continue
		}
		localUIDs[uid] = struct{}{}
	}

	gClient, err := grafana.ClientFromContext(cfg)
	if err != nil {
		return 0, 0, err
	}

	listResp, err := gClient.Provisioning.GetAlertRulesWithParams(provisioning.NewGetAlertRulesParams().WithContext(ctx))
	if err != nil {
		return 0, 0, err
	}

	for _, r := range listResp.Payload {
		uid := strings.TrimSpace(r.UID)
		if uid == "" {
			continue
		}

		if _, ok := localUIDs[uid]; ok {
			continue
		}

		if dryRun {
			deleted++
			continue
		}

		if _, err := gClient.Provisioning.DeleteAlertRule(provisioning.NewDeleteAlertRuleParams().WithUID(uid).WithContext(ctx)); err != nil {
			failed++
			if stopOnError {
				return deleted, failed, err
			}
			continue
		}

		deleted++
	}

	return deleted, failed, nil
}

func readAlertRuleFiles(paths []string) ([]*models.ProvisionedAlertRule, error) {
	var rules []*models.ProvisionedAlertRule

	for _, root := range paths {
		// We only consider files in <path>/Alerts/.
		alertsDir := filepath.Join(root, alertsDirName)
		info, err := os.Stat(alertsDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if !info.IsDir() {
			continue
		}

		entries, err := os.ReadDir(alertsDir)
		if err != nil {
			return nil, err
		}

		for _, e := range entries {
			if e.IsDir() {
				continue
			}

			name := e.Name()
			ext := strings.TrimPrefix(filepath.Ext(name), ".")
			var decoder format.Codec
			switch ext {
			case "json":
				decoder = format.NewJSONCodec()
			case "yaml", "yml":
				decoder = format.NewYAMLCodec()
			default:
				continue
			}

			full := filepath.Join(alertsDir, name)
			f, err := os.Open(full)
			if err != nil {
				return nil, err
			}

			var rule models.ProvisionedAlertRule
			if err := decoder.Decode(f, &rule); err != nil {
				_ = f.Close()
				return nil, fmt.Errorf("parse error in '%s': %w", full, err)
			}
			_ = f.Close()

			// If uid is missing in file, fall back to filename stem.
			if strings.TrimSpace(rule.UID) == "" {
				rule.UID = strings.TrimSuffix(name, filepath.Ext(name))
			}

			rules = append(rules, &rule)
		}
	}

	return rules, nil
}

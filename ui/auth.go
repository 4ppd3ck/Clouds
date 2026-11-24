package ui

import (
	"fmt"
	"os"
	"strings"

	"clouds/cloud"
	"clouds/providers"
)

type providerFailure struct {
	Message string
	Next    string
	Command string
}

func authContent(connections []providers.ConnectionInfo, failures map[string]providerFailure, opts cloud.Options) string {
	var b strings.Builder
	b.WriteString(" CONNECT / AUTH\n\n")
	b.WriteString("Credentials use the standard AWS and Google provider chains: local config, SSO, environment, or instance identity. Clouds never asks for or stores secret keys.\n\n")
	for _, provider := range connections {
		fmt.Fprintf(&b, " %s  [%s]\n", strings.ToUpper(provider.Provider), strings.ToUpper(provider.State))
		fmt.Fprintf(&b, "   %s\n", provider.Summary)
		if provider.Detail != "" {
			fmt.Fprintf(&b, "   %s\n", provider.Detail)
		}
		if provider.Next != "" {
			fmt.Fprintf(&b, "   Next: %s\n", provider.Next)
		}
		if provider.Command != "" {
			fmt.Fprintf(&b, "   Run: %s\n", provider.Command)
		}
		if failure, ok := failures[provider.Provider]; ok {
			fmt.Fprintf(&b, "   Last request: %s\n   Recovery: %s\n", failure.Message, failure.Next)
			if failure.Command != "" {
				fmt.Fprintf(&b, "   Try: %s\n", failure.Command)
			}
		}
		b.WriteByte('\n')
	}
	profile := opts.Profile
	if profile == "" {
		profile = os.Getenv("AWS_PROFILE")
		if profile == "" {
			profile = "default"
		}
	}
	fmt.Fprintf(&b, " AWS profile: %s\n", profile)
	b.WriteString(" GCP project: set with --project or `project = ...` in the Clouds config.\n\n")
	b.WriteString(" After changing credentials, restart Clouds to reload them. Press r for transient request failures. For temporary exports, run `clouds` directly in that same terminal; a Finder-launched app or another terminal will not inherit them.\n\n")
	b.WriteString(" Keys: r recheck connections · esc return · ? help\n")
	return b.String()
}

func compactConnectionSummary(conn providers.ConnectionInfo) string {
	if conn.State != "ready" && conn.State != "demo" {
		return cloud.Trunc(conn.Summary, 30)
	}
	if conn.State == "demo" {
		return "sample data"
	}
	fields := strings.Fields(conn.Summary)
	switch conn.Provider {
	case "aws":
		profile, account := "default", "unknown"
		for _, field := range fields {
			if strings.HasPrefix(field, "profile=") {
				profile = strings.TrimPrefix(field, "profile=")
			}
			if strings.HasPrefix(field, "account=") {
				account = strings.TrimPrefix(field, "account=")
			}
		}
		return "profile=" + cloud.Trunc(profile, 12) + " acct=" + account
	case "gcp":
		for _, field := range fields {
			if strings.HasPrefix(field, "project=") {
				return cloud.Trunc(field, 30)
			}
		}
	}
	return cloud.Trunc(conn.Summary, 30)
}

func explainProviderFailure(providerID string, err error, opts cloud.Options) providerFailure {
	message := err.Error()
	lower := strings.ToLower(message)
	switch providerID {
	case "aws":
		profile := opts.Profile
		if profile == "" {
			profile = os.Getenv("AWS_PROFILE")
			if profile == "" {
				profile = "default"
			}
		}
		if strings.Contains(lower, "credential") || strings.Contains(lower, "token") || strings.Contains(lower, "sso") || strings.Contains(lower, "expired") {
			return providerFailure{Message: message, Next: "Refresh the selected AWS login, then restart Clouds to reload credentials. Restart from the same shell if you use exported variables.", Command: "aws sso login --profile " + profile + "   (SSO)\naws configure --profile " + profile + "   (other profiles)"}
		}
		if strings.Contains(lower, "accessdenied") || strings.Contains(lower, "access denied") || strings.Contains(lower, "unauthorizedoperation") || strings.Contains(lower, "403") {
			return providerFailure{Message: message, Next: "The AWS identity was found, but it may lack permission for this service. Check its IAM policy and the selected region."}
		}
		return providerFailure{Message: message, Next: "Check network access, the selected region, and the AWS service permissions. Press r to retry."}
	case "gcp":
		if strings.Contains(lower, "no gcp project") || strings.Contains(lower, "project") && strings.Contains(lower, "required") {
			return providerFailure{Message: message, Next: "Set the project in Connect/Auth or pass --project, then retry.", Command: "clouds --project YOUR_PROJECT_ID"}
		}
		if strings.Contains(lower, "credential") || strings.Contains(lower, "oauth") || strings.Contains(lower, "401") || strings.Contains(lower, "unauthenticated") {
			return providerFailure{Message: message, Next: "Refresh Application Default Credentials, then restart Clouds to reload credentials.", Command: "gcloud auth application-default login"}
		}
		if strings.Contains(lower, "permissiondenied") || strings.Contains(lower, "permission denied") || strings.Contains(lower, "403") {
			return providerFailure{Message: message, Next: "Credentials were accepted, but the account may lack this service's IAM role or the API may be disabled."}
		}
		return providerFailure{Message: message, Next: "Check the GCP project, service API, IAM access, and network, then press r to retry."}
	default:
		return providerFailure{Message: message, Next: "Check the provider setup, then retry."}
	}
}

package ui

import (
	"fmt"
	"strings"

	"clouds/cloud"
)

// helpContent renders the help screen. The service lists are derived from the
// active catalogs rather than hardcoded, so the help screen can never drift
// from what the binary actually supports (in --demo mode it lists exactly the
// demo services that are really available).
func helpContent(catalogs []cloud.Provider) string {
	var b strings.Builder
	fmt.Fprintf(&b, ` clouds v%s — k9s-style browser for AWS & GCP

 NAVIGATION
  enter        open detail (or drill down: S3 objects, ECS services, log events)
  esc / q      back — q at the top level quits
  :<service>   jump to a service (see list below), e.g. :ec2 :s3 :lambda
  :aws/:gcp    switch cloud provider
  /            filter rows (live substring match, esc to clear)
  s / S        sort — s cycles the sort column, S flips ascending vs descending
  p            toggle provider (aws <-> gcp)
  1 / 2        jump to aws / gcp
  r            refresh the current view
  d            open the selected resource in the web console
  g            download the selected S3/GCS object to disk (folders: recursive)
  y            copy the selected resource name to the clipboard
  ?            this help
  :q           quit

`, Version)
	for _, p := range catalogs {
		fmt.Fprintf(&b, " %s SERVICES\n%s\n\n", strings.ToUpper(p.ID), serviceLine(p))
	}
	b.WriteString(` AUTH
  AWS  default credential chain (~/.aws, env vars, SSO, IMDS).
       Overrides: --profile, --region
  GCP  Application Default Credentials — run: gcloud auth application-default login
       Project: --project flag or GOOGLE_CLOUD_PROJECT env var
`)
	return b.String()
}

// serviceLine renders one provider's services as ":id" tokens, appending the
// alias in parentheses when the service has one.
func serviceLine(p cloud.Provider) string {
	tokens := make([]string, 0, len(p.Services))
	for _, s := range p.Services {
		tok := ":" + s.ID
		if s.Alias != "" {
			tok += " (" + s.Alias + ")"
		}
		tokens = append(tokens, tok)
	}
	return "  " + strings.Join(tokens, "  ")
}

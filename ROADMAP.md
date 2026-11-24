# Clouds Roadmap

This roadmap tracks candidate work for future releases. Items are ordered by
area, not by delivery date; scope and priority may change as Clouds evolves.

## Planned

### Connect to compute instances

- Add SSH access to EC2 instances from the resource view, using a local SSH
  client and user-selected login name/key. Never upload or store private keys.
- Add AWS Systems Manager Session Manager access where the instance and AWS
  identity are configured for it.
- Add `gcloud compute ssh` access for Google Compute Engine instances.
- Show clear reasons when a connection cannot start, such as no reachable
  address, missing local key, unavailable SSH client, or missing SSM setup.

### Logs and operations

- Add live log following for CloudWatch Logs and Google Cloud Logging.
- Add carefully scoped resource actions, starting with EC2 start/stop and Cloud
  Run restart. Show the target and require confirmation before mutations.

### Navigation and export

- Add saved views and bookmarks for frequently used services and filters.
- Add YAML export for resource lists and details.

### Provider support

- Add an Azure provider with the same connection summary, resource browsing,
  detail, and error handling patterns used for AWS and GCP.

### Release and installation follow-up

- Publish the local installers in the next release ZIPs and verify that they
  configure command access from zsh, bash, and Windows PowerShell.
- Check release ZIP contents, checksums, and launch behavior for every supported
  operating system and architecture before publishing.

## Recently completed

- Add a Connect/Auth screen with AWS and GCP credential status and recovery
  guidance.
- Let users choose and save the download folder.
- Browse S3 and GCS objects one folder level at a time.
- Allow native terminal text selection by leaving mouse reporting disabled.
- Normalize the displayed version so it has one `v` prefix.

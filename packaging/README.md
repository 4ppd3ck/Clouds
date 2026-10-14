# packaging — third-party package manager manifests

Templates for distributing `clouds` beyond the GitHub releases page. The
repo/owner in every URL defaults to `GITHUB_REPO/clouds` — find-and-replace
if you rename the app or host it elsewhere.

| File | For | Status |
|------|-----|--------|
| `clouds.json` | [Scoop](https://scoop.sh) (Windows) | fill `hash` fields from the release's `SHA256SUMS` |
| `clouds.rb` | [Homebrew](https://brew.sh) (macOS/Linux) | fill `sha256` fields from the release's `SHA256SUMS` |

## Scoop (Windows)

1. Create a bucket repo, e.g. `GITHUB_REPO/scoop-bucket`, with `clouds.json`
   in its root.
2. Per release, update `version` and each `hash` (from `SHA256SUMS`). The
   `autoupdate` block lets tools like `checkver` automate this.
3. Users install with:

   ```powershell
   scoop bucket add GITHUB_REPO https://github.com/GITHUB_REPO/scoop-bucket
   scoop install clouds
   ```

## Homebrew (macOS / Linux)

1. Create a tap repo, e.g. `GITHUB_REPO/homebrew-tap`, with this file at
   `Formula/clouds.rb`.
2. Per release, update `version` and the four `sha256` values.
3. Users install with:

   ```bash
   brew tap GITHUB_REPO/tap
   brew install clouds
   ```

Tip: a small CI job in the tap repo can watch releases and regenerate
hashes automatically (`gh release view` + `SHA256SUMS`).

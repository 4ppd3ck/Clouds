# Homebrew formula template. To publish:
#   1. Create a tap repo, e.g. GITHUB_REPO/homebrew-tap
#   2. Copy this file there as Formula/clouds.rb
#   3. Fill in the real sha256 values from the release's SHA256SUMS
#      (they change every release — `brew tap-new` + autoupdate can automate)
# Users then install with:
#   brew tap GITHUB_REPO/tap && brew install clouds
class Clouds < Formula
  desc "k9s-style terminal UI for browsing AWS and GCP resources"
  homepage "https://github.com/GITHUB_REPO/clouds"
  version "0.1.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/GITHUB_REPO/clouds/releases/download/v0.1.0/cloud-darwin-arm64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
    on_intel do
      url "https://github.com/GITHUB_REPO/clouds/releases/download/v0.1.0/cloud-darwin-amd64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/GITHUB_REPO/clouds/releases/download/v0.1.0/cloud-linux-arm64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
    on_intel do
      url "https://github.com/GITHUB_REPO/clouds/releases/download/v0.1.0/cloud-linux-amd64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end

  def install
    binary = Dir["cloud-*"].first
    odie "unexpected archive layout" if binary.nil?
    bin.install binary => "clouds"
  end

  test do
    assert_match "clouds", shell_output("#{bin}/clouds --version")
  end
end

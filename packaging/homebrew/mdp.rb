# Homebrew formula for mdp
# To use: create a tap repo (e.g., donaldgifford/homebrew-tap)
# and add this formula, then: brew install donaldgifford/tap/mdp
class Mdp < Formula
  desc "Fast markdown preview server for Neovim with live reload and scroll sync"
  homepage "https://github.com/donaldgifford/mdp"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/donaldgifford/mdp/releases/download/v#{version}/mdp_darwin_arm64.tar.gz"
    else
      url "https://github.com/donaldgifford/mdp/releases/download/v#{version}/mdp_darwin_amd64.tar.gz"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/donaldgifford/mdp/releases/download/v#{version}/mdp_linux_arm64.tar.gz"
    else
      url "https://github.com/donaldgifford/mdp/releases/download/v#{version}/mdp_linux_amd64.tar.gz"
    end
  end

  def install
    bin.install "mdp"
  end

  test do
    assert_match "mdp version", shell_output("#{bin}/mdp --version")
  end
end

class Sqliter < Formula
  desc "SQLite viewer and editor"
  homepage "https://github.com/darianmavgo/sqliter"
  head "https://github.com/darianmavgo/sqliter.git", branch: "main"
  license "MIT"

  depends_on "go" => :build
  depends_on "node" => :build

  def install
    # Build React client
    cd "react-client" do
      system "npm", "install"
      system "npm", "run", "build"
    end

    # Copy built assets to embed directory
    mkdir_p "sqliter/ui"
    cp_r "react-client/dist/.", "sqliter/ui"

    # Build Go binary
    ldflags = "-X main.version=#{version}"
    system "go", "build", *std_go_args(ldflags: ldflags), "./cmd/sqliter"
  end

  test do
    assert_match "sqliter version", shell_output("#{bin}/sqliter --version")
  end
end

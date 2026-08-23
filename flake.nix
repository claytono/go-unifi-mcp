{
  description = "go-unifi-mcp development environment";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" "aarch64-darwin" "x86_64-darwin" "aarch64-linux" ];
      forEachSupportedSystem = f: nixpkgs.lib.genAttrs supportedSystems (system: f {
        pkgs = import nixpkgs { inherit system; };
      });

      mkGoUnifiMcp = pkgs:
        let
          version = if self ? rev then self.rev else "dirty";
        in
        pkgs.buildGoModule {
          pname = "go-unifi-mcp";
          inherit version;

          src = self;
          subPackages = [ "cmd/go-unifi-mcp" ];

          ldflags = [
            "-s"
            "-w"
            "-X github.com/claytono/go-unifi-mcp/internal/server.Version=${version}"
          ];

          vendorHash = "sha256-i43KDHi95dPLinhzeyvXFWI0KBcAUY4TlADRlnltQoY=";
          goSum = ./go.sum;

          meta = with pkgs.lib; {
            description = "MCP server for UniFi Network Controller";
            homepage = "https://github.com/claytono/go-unifi-mcp";
            license = licenses.mpl20;
            mainProgram = "go-unifi-mcp";
          };
        };

      # go-test-coverage package (not in nixpkgs)
      mkGoTestCoverage = pkgs: pkgs.buildGoModule rec {
        pname = "go-test-coverage";
        version = "2.19.0";

        src = pkgs.fetchFromGitHub {
          owner = "vladopajic";
          repo = "go-test-coverage";
          rev = "v${version}";
          name = "go-test-coverage-${version}-source";
          hash = "sha256-3tQnOND1ChzZFgh6UX9ahlvh4CNLvytzLAMh9eIgDQc=";
        };

        vendorHash = "sha256-yqYRnfGjd6iDsQTec1MQPYY58IM8FekxDzSNl3UaP28=";

        passthru.refreshSourceHash = true;

        # Skip tests - upstream integration tests require GitHub credentials (#243)
        doCheck = false;
        checkPhase = "";

        meta = {
          description = "Tool to report issues when test coverage falls below threshold";
          homepage = "https://github.com/vladopajic/go-test-coverage";
        };
      };

      # python-kacl for changelog validation and extraction
      mkPythonKacl = pkgs: pkgs.python3Packages.buildPythonApplication rec {
        pname = "python-kacl";
        version = "0.7.3";
        pyproject = true;

        src = pkgs.fetchFromGitLab {
          owner = "schmieder.matthias";
          repo = "python-kacl";
          rev = "v${version}";
          name = "python-kacl-${version}-source";
          hash = "sha256-xie0cYp8b63N5BR4l99Tsct+Yrisw+kYeowpCeFxP7Y=";
        };

        build-system = with pkgs.python3Packages; [ setuptools ];

        dependencies = with pkgs.python3Packages; [
          click
          semver
          gitpython
          pyyaml
          jira
        ];

        doCheck = false;

        passthru.refreshSourceHash = true;

        meta = {
          description = "CLI tool to manage changelogs in Keep a Changelog format";
          homepage = "https://gitlab.com/schmieder.matthias/python-kacl";
        };
      };

      # mcp-publisher for publishing to MCP Registry
      mkMcpPublisher = pkgs: let
        version = "1.8.0";
        sources = {
          "aarch64-darwin" = {
            url = "https://github.com/modelcontextprotocol/registry/releases/download/v${version}/mcp-publisher_darwin_arm64.tar.gz";
            hash = "sha256-50+IRsO10EKM/q4/n1ILv5Ax0Y5oIkEIw3YNYLaq8uA=";
          };
          "x86_64-darwin" = {
            url = "https://github.com/modelcontextprotocol/registry/releases/download/v${version}/mcp-publisher_darwin_amd64.tar.gz";
            hash = "sha256-U1D3VuhAjQ4igCt/OEr5QUSDWLUD6x4XcpeaYbm5n94=";
          };
          "x86_64-linux" = {
            url = "https://github.com/modelcontextprotocol/registry/releases/download/v${version}/mcp-publisher_linux_amd64.tar.gz";
            hash = "sha256-E3BEa7501WJgjoAFpszOAtFGpmH714Z04RzHC5YY1s8=";
          };
          "aarch64-linux" = {
            url = "https://github.com/modelcontextprotocol/registry/releases/download/v${version}/mcp-publisher_linux_arm64.tar.gz";
            hash = "sha256-yXiYLGDhtJA6l23gkPBNxPrEoyDapQcE/K0tvJNDPWI=";
          };
        };
        src = sources.${pkgs.stdenv.hostPlatform.system} or null;
      in if src == null then null else pkgs.stdenv.mkDerivation {
        pname = "mcp-publisher";
        inherit version;

        src = pkgs.fetchurl {
          inherit (src) url hash;
          name = "mcp-publisher-${version}-${pkgs.stdenv.hostPlatform.system}.tar.gz";
        };

        sourceRoot = ".";
        unpackPhase = ''
          tar xzf $src
        '';

        installPhase = ''
          mkdir -p $out/bin
          cp mcp-publisher $out/bin/
          chmod +x $out/bin/mcp-publisher
        '';

        passthru.refreshSourceHash = true;

        meta = {
          description = "CLI tool for publishing servers to the MCP Registry";
          homepage = "https://github.com/modelcontextprotocol/registry";
        };
      };

      # mcp-cli for invoking MCP servers from CLI
      # Returns null on unsupported platforms (aarch64-linux has no binary)
      mkMcpCli = pkgs: let
        version = "0.3.0";
        sources = {
          "aarch64-darwin" = {
            url = "https://github.com/philschmid/mcp-cli/releases/download/v${version}/mcp-cli-darwin-arm64";
            hash = "sha256-vpkd8KEl4c+aAi/m/84jZgVSL9E4JDXOWP/fmrpmQvI=";
          };
          "x86_64-darwin" = {
            url = "https://github.com/philschmid/mcp-cli/releases/download/v${version}/mcp-cli-darwin-x64";
            hash = "sha256-8OiQpmYmgzUAW7uhmSU/+Pbmr6520K3zT72nCLJqzC4=";
          };
          "x86_64-linux" = {
            url = "https://github.com/philschmid/mcp-cli/releases/download/v${version}/mcp-cli-linux-x64";
            hash = "sha256-dncvKQ7aqFbL7JZ9EsM8ub9Jz/AU9Vow0EJFz4lwgXw=";
          };
        };
        src = sources.${pkgs.stdenv.hostPlatform.system} or null;
      in if src == null then null else pkgs.stdenv.mkDerivation {
        pname = "mcp-cli";
        inherit version;

        src = pkgs.fetchurl {
          inherit (src) url hash;
          name = "mcp-cli-${version}-${pkgs.stdenv.hostPlatform.system}";
        };

        dontUnpack = true;

        installPhase = ''
          mkdir -p $out/bin
          cp $src $out/bin/mcp-cli
          chmod +x $out/bin/mcp-cli
        '';

        passthru.refreshSourceHash = true;

        meta = {
          description = "Lightweight CLI for interacting with MCP servers";
          homepage = "https://github.com/philschmid/mcp-cli";
          platforms = [ "aarch64-darwin" "x86_64-darwin" "x86_64-linux" ];
        };
      };
    in
    {
      packages = forEachSupportedSystem ({ pkgs }: {
        default = mkGoUnifiMcp pkgs;
      });

      apps = forEachSupportedSystem ({ pkgs }: {
        default = {
          type = "app";
          program = "${mkGoUnifiMcp pkgs}/bin/go-unifi-mcp";
        };
      });

      devShells = forEachSupportedSystem ({ pkgs }: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            go_1_26
            golangci-lint
            go-task
            pre-commit
            goreleaser
            go-mockery
            (mkGoTestCoverage pkgs)
            (mkPythonKacl pkgs)
          ] ++ lib.optional (mkMcpCli pkgs != null) (mkMcpCli pkgs)
            ++ lib.optional (mkMcpPublisher pkgs != null) (mkMcpPublisher pkgs);
        };
      });
    };
}

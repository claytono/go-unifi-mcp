{
  description = "go-unifi-mcp development environment";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" "aarch64-darwin" "aarch64-linux" ];
      forEachSupportedSystem = f: nixpkgs.lib.genAttrs supportedSystems (system: f {
        pkgs = import nixpkgs { inherit system; };
      });

      # One Go toolchain for the dev shell and every Go build. Renovate bumps
      # the attribute below; nixpkgs' default go lags new Go releases.
      goFor = pkgs: pkgs.go_1_27;
      buildGoModuleFor = pkgs: pkgs.buildGoModule.override { go = goFor pkgs; };

      mkGoUnifiMcp = pkgs:
        let
          version = if self ? rev then self.rev else "dirty";
        in
        (buildGoModuleFor pkgs) {
          pname = "go-unifi-mcp";
          inherit version;

          src = self;
          subPackages = [ "cmd/go-unifi-mcp" ];

          ldflags = [
            "-s"
            "-w"
            "-X github.com/claytono/go-unifi-mcp/internal/server.Version=${version}"
          ];

          vendorHash = "sha256-VYsLzejwSwhaCmzXW1T2IV2CdVUQC9RG1EGHYR9CjTo=";
          goSum = ./go.sum;

          meta = with pkgs.lib; {
            description = "MCP server for UniFi Network Controller";
            homepage = "https://github.com/claytono/go-unifi-mcp";
            license = licenses.mpl20;
            mainProgram = "go-unifi-mcp";
          };
        };

      # go-test-coverage package (not in nixpkgs)
      mkGoTestCoverage = pkgs: (buildGoModuleFor pkgs) rec {
        pname = "go-test-coverage";
        version = "2.20.1";

        src = pkgs.fetchFromGitHub {
          owner = "vladopajic";
          repo = "go-test-coverage";
          rev = "v${version}";
          name = "go-test-coverage-${version}-source";
          hash = "sha256-lSQc8cRL4ZkFaZAcqzwlrNO5Gq+4uQzb9Gkw9tgumm0=";
        };

        vendorHash = "sha256-UO7gVUIcQectjZWCiN3AfF2ELgjHn7UCZTnNtG37Gxg=";

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
        version = "0.7.7";
        pyproject = true;

        src = pkgs.fetchFromGitLab {
          owner = "schmieder.matthias";
          repo = "python-kacl";
          rev = "v${version}";
          name = "python-kacl-${version}-source";
          hash = "sha256-KwsjknnwyJHA0pQZI4R30YgyesCGIrZqe5cbM/4dnV8=";
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
        version = "1.8.6";
        sources = {
          "aarch64-darwin" = {
            url = "https://github.com/modelcontextprotocol/registry/releases/download/v${version}/mcp-publisher_darwin_arm64.tar.gz";
            hash = "sha256-lsKIb5GdrlrE5hhAcCVu9DLkhfEnST89wFKvELOcIbA=";
          };
          "x86_64-linux" = {
            url = "https://github.com/modelcontextprotocol/registry/releases/download/v${version}/mcp-publisher_linux_amd64.tar.gz";
            hash = "sha256-XcL9QVNpxzVRJZ8xzTPiqsyeDRAh5/CP7xSS2Tt4Fig=";
          };
          "aarch64-linux" = {
            url = "https://github.com/modelcontextprotocol/registry/releases/download/v${version}/mcp-publisher_linux_arm64.tar.gz";
            hash = "sha256-46lD0OfS8K5RR5Bdp1EpixHSE1kwuemBpJeyQRL+DhI=";
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
          platforms = [ "aarch64-darwin" "x86_64-linux" ];
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
            (goFor pkgs)
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

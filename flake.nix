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

          vendorHash = "sha256-jD+R9IGusJBLIWmfsPPHulQoED+cXEMlaixFAgQLIms=";

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
        version = "2.18.3";

        src = pkgs.fetchFromGitHub {
          owner = "vladopajic";
          repo = "go-test-coverage";
          rev = "v${version}";
          hash = "sha256-8KPnufCLGR3beBjTJSGSkxZd+m3r1pYDtTBLhG/eSEg=";
        };

        vendorHash = "sha256-iJ3VFnzPYd0ovyK/QdCDolh5p8fe/aXulnHxAia5UuE=";

        # Skip tests entirely - upstream has two issues:
        # 1. Integration tests that require GitHub credentials (#243)
        # 2. Nil pointer bug in Test_Github_Error with Go 1.25 (#266)
        doCheck = false;
        checkPhase = "";

        meta = {
          description = "Tool to report issues when test coverage falls below threshold";
          homepage = "https://github.com/vladopajic/go-test-coverage";
        };
      };

      # mcp-cli for invoking MCP servers from CLI
      # Returns null on unsupported platforms (aarch64-linux has no binary)
      mkMcpCli = pkgs: let
        version = "0.1.4";
        sources = {
          "aarch64-darwin" = {
            url = "https://github.com/philschmid/mcp-cli/releases/download/v${version}/mcp-cli-darwin-arm64";
            hash = "sha256-WNKFzfHbCgA2TGqHJ3XOJKUKW+kE4kdexlTQ/BYH2PY=";
          };
          "x86_64-darwin" = {
            url = "https://github.com/philschmid/mcp-cli/releases/download/v${version}/mcp-cli-darwin-x64";
            hash = "sha256-KSC5atyBKVKGZZTYlxVrR9r4fHR3ynr35bN0Fouz1NI=";
          };
          "x86_64-linux" = {
            url = "https://github.com/philschmid/mcp-cli/releases/download/v${version}/mcp-cli-linux-x64";
            hash = "sha256-nPfQOEyp1wR/KgHsUILIL3M/epkEpwePZ8TiHOTiHCQ=";
          };
        };
        src = sources.${pkgs.stdenv.hostPlatform.system} or null;
      in if src == null then null else pkgs.stdenv.mkDerivation {
        pname = "mcp-cli";
        inherit version;

        src = pkgs.fetchurl {
          inherit (src) url hash;
        };

        dontUnpack = true;

        installPhase = ''
          mkdir -p $out/bin
          cp $src $out/bin/mcp-cli
          chmod +x $out/bin/mcp-cli
        '';

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
            go_1_25
            golangci-lint
            go-task
            pre-commit
            goreleaser
            go-mockery
            (mkGoTestCoverage pkgs)
          ] ++ lib.optional (mkMcpCli pkgs != null) (mkMcpCli pkgs);
        };
      });
    };
}

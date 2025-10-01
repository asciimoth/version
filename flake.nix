# Usage:
#   nix profile add github:github.com/asciimoth/version
#   nix profile remove version
#   nix shell github:github.com/asciimoth/version
# Update: nix flake update
{
  description = "Multi-source semantic version management tool";
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    pre-commit-hooks.url = "github:cachix/pre-commit-hooks.nix";
    gomod2nix = {
      url = "github:tweag/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-utils.follows = "flake-utils";
    };
  };
  outputs = {
    self,
    nixpkgs,
    flake-utils,
    pre-commit-hooks,
    gomod2nix,
    ...
  }:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs {
        inherit system;
        overlays = [ gomod2nix.overlays.default ];
      };

      app = pkgs.buildGoApplication {
        name = "version";
        version = builtins.readFile ./VERSION;
        src = ./.;
        modules = ./gomod2nix.toml;
        nativeBuildInputs = [ pkgs.gzip ];
        # after the default install, put the man into $out/share/man/man1/
        postInstall = ''
          mkdir -p $out/share/man/man1
          if [ -f ${./man/version.1} ]; then
            gzip -n -c -k ${./man/version.1} > $out/share/man/man1/version.1.gz
          fi
        '';
      };

      release = pkgs.writeShellScriptBin "release" (builtins.readFile ./ci/release);

      checks = {
        pre-commit-check = pre-commit-hooks.lib.${system}.run {
          src = ./.;
          hooks = {
            commitizen.enable = true;
            typos.enable = true;
            typos-commit = {
              enable = true;
              description = "Find typos in commit message";
              entry = let script = pkgs.writeShellScript "typos-commit" ''
                typos "$1"
              ''; in builtins.toString script;
              stages = [ "commit-msg" ];
            };
            govet.enable = true;
            gofmt.enable = true;
            golangci-lint.enable = true;
            gotidy = {
              enable = true;
              description = "Makes sure go.mod matches the source code";
              entry = let script = pkgs.writeShellScript "gotidyhook" ''
                go mod tidy -v
                if [ -f "go.mod" ]; then
                  git add go.mod
                fi
                if [ -f "go.sum" ]; then
                  git add go.sum
                fi
              ''; in builtins.toString script;
              stages = [ "pre-commit" ];
            };
            gomod2nix = {
              enable = true;
              description = "Generates gomod2nix.toml";
              entry = let script = pkgs.writeShellScript "gomod2nix-hook" ''
                gomod2nix generate
                git add gomod2nix.toml
              ''; in builtins.toString script;
              stages = [ "pre-commit" ];
              after = [ "gotidy" ];
            };
          };
        };
      };
    in {
      devShells.default = pkgs.mkShell {
        inherit (checks.pre-commit-check) shellHook;
        buildInputs = with pkgs; [
          go
          golangci-lint
          commitizen
          goreleaser
          git-cliff
          govulncheck

          gomod2nix.packages.${system}.default

          typos

          yq
          jq

          release
        ];
      };

      packages.default = app;
    });
}

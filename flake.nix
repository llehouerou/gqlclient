{
  description = "gqlclient development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { nixpkgs, ... }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    {
      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [
              # Go toolchain
              go
              gopls
              golines
              goimports-reviser
              golangci-lint
              delve

              # Build tools
              gnumake

              # Nix
              nil # Nix language server
            ];

            shellHook = ''
              export GOPATH="$HOME/go"
              export PATH="$GOPATH/bin:$PATH"
              # Default to CGO off; override (CGO_ENABLED=1) when running
              # the race detector or anything else that needs cgo.
              export CGO_ENABLED=0
            '';
          };
        }
      );
    };
}

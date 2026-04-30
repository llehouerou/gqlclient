{
  description = "gqlclient development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { nixpkgs, ... }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
    in
    {
      devShells.${system}.default = pkgs.mkShell {
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
          export CGO_ENABLED=0
        '';
      };
    };
}

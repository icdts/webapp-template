{
  description = "Learning HTMX";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" ];
      forAllSystems =
        f: nixpkgs.lib.genAttrs supportedSystems (system: f (import nixpkgs { inherit system; }));
    in
    {
      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {

          packages = with pkgs; [
            go
            gopls
            gotools
            air

            gnumake

            gnupg
            podman

            sqlite
            pkg-config
            postgresql
          ];

          shellHook =
            let
              htmxFile = pkgs.fetchurl {
                url = "https://cdn.jsdelivr.net/npm/htmx.org@2.0.8/dist/htmx.min.js";
                sha256 = "04qksd80lz91ap9c306ms41sfmkjvkg1p2m8y0a5jm5pikv3wa12";
              };
              redhatKey = pkgs.fetchurl {
                url = "https://security.access.redhat.com/data/fd431d51.txt";
                sha256 = "0avlzkx44v7iaj4fsyiwi5rc04p7m8qig8yl72a1vqxy4qv59cnl"; 
              };
            in
            ''
              echo "Go version $(go version)"
              export HTMX_SRC=${htmxFile}
              echo "HTMX_SRC -> $HTMX_SRC"
              export REDHAT_GPG="${redhatKey}"
              echo "REDHAT_GPG -> $REDHAT_GPG"
            '';
        };
      });
    };
}

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
          ];

          shellHook =
            let
              htmxFile = pkgs.fetchurl {
                url = "https://cdn.jsdelivr.net/npm/htmx.org@2.0.8/dist/htmx.min.js";
                sha256 = "04qksd80lz91ap9c306ms41sfmkjvkg1p2m8y0a5jm5pikv3wa12";
              };
            in
            ''
              echo "Go version $(go version)"
              export HTMX_SRC=${htmxFile}
              echo "HTMX_SRC -> $HTMX_SRC"
            '';
        };
      });
    };
}

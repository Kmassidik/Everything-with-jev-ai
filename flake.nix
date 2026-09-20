{
  description = "jevai-products — Jev-powered judgment platform (Go · templ · HTMX · Tailwind)";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "aarch64-darwin" "x86_64-darwin" "aarch64-linux" "x86_64-linux" ];
      forAll = f: nixpkgs.lib.genAttrs systems (s: f nixpkgs.legacyPackages.${s});
    in {
      devShells = forAll (pkgs: {
        default = pkgs.mkShell {
          # Everything the build needs — no Node/JS toolchain anywhere.
          packages = with pkgs; [ go templ tailwindcss air just golangci-lint gofumpt sqlite ];
          shellHook = ''
            echo "jevai-products · go $(go version | cut -d' ' -f3) · templ $(templ version)"
            echo "→ just dev"
          '';
        };
      });
    };
}

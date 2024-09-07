{
  description = "Tagesschau Breaking News bot for Telegram";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixpkgs-unstable";
  };

  outputs = { self, nixpkgs, ... }:

    let
      forAllSystems = function:
        nixpkgs.lib.genAttrs [
          "x86_64-linux"
          "aarch64-linux"
          "x86_64-darwin"
          "aarch64-darwin"
        ]
          (system: function nixpkgs.legacyPackages.${system});

      version =
        if (self ? shortRev)
        then self.shortRev
        else "dev";
    in
    {

      nixosModules = {
        default = ./module.nix;
      };

      overlays.default = final: prev: {
        tagesschau-eilbot = self.packages.${prev.system}.default;
      };

      devShells = forAllSystems
        (pkgs: {
          default = pkgs.mkShell {
            packages = [
              pkgs.go
              pkgs.golangci-lint
            ];
            shellHook = ''
              export DEBUG=1
            '';
          };
        });


      packages = forAllSystems
        (pkgs: {
          tagesschau-eilbot =
            pkgs.buildGoModule
              {
                pname = "tagesschau-eilbot";
                inherit version;
                src = pkgs.lib.cleanSource self;

                # Update the hash if go dependencies change!
                # vendorHash = pkgs.lib.fakeHash;
                vendorHash = "sha256-pSmwUsyI0mLi7GdF5kpXW0QRXp/EnGGwifFfa9w/Y2w=";

                ldflags = [ "-s" "-w" ];

                meta = {
                  description = "Tagesschau Breaking News bot for Telegram";
                  homepage = "https://github.com/Brawl345/tagesschau-eilbot";
                  license = pkgs.lib.licenses.unlicense;
                  platforms = pkgs.lib.platforms.darwin ++ pkgs.lib.platforms.linux;
                };
              };

          default = self.packages.${pkgs.system}.tagesschau-eilbot;
        });
    };
}

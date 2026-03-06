{
  description = "A flake for grlx-nom, with Hercules CI support";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

    crane = {
      url = "github:ipetkov/crane";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    flake-utils.url = "github:numtide/flake-utils";
    flake-parts.url = "github:hercules-ci/flake-parts";
  };

  outputs = inputs @ {
    self,
    flake-parts,
    nixpkgs,
    crane,
    flake-utils,
    ...
  }:
    flake-parts.lib.mkFlake {inherit inputs;} {
      systems = ["x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin"];
      perSystem = {
        pkgs,
        system,
        ...
      }: let
        pkgs = import nixpkgs {
          inherit system;
        };
        craneLib = crane.lib.${system};
        commonArgs = {
          src = craneLib.cleanCargoSource (craneLib.path ./.);
          buildInputs = with pkgs;
            [
              openssl
              pkg-config
              # Add additional build inputs here
            ]
            ++ lib.optionals stdenv.isDarwin [
              # Additional darwin specific inputs can be set here
              libiconv
            ];
        };
        cargoArtifacts = craneLib.buildDepsOnly (commonArgs
          // {
            pname = "grlx-nom-deps";
          });

        grlx-nom-clippy = craneLib.cargoClippy (commonArgs
          // {
            inherit cargoArtifacts;
            cargoClippyExtraArgs = "--all-targets -- --deny warnings";
          });

        grlx-nom-nextest = craneLib.cargoNextest (commonArgs
          // {
            inherit cargoArtifacts;
          });

        grlx-nom = craneLib.buildPackage (commonArgs
          // {
            inherit cargoArtifacts;
          });
      in {
        checks = {
          inherit
            grlx-nom
            grlx-nom-clippy
            grlx-nom-nextest
            ;
        };
        formatter = pkgs.alejandra;
        packages.default = grlx-nom;

        apps.default = flake-utils.lib.mkApp {
          drv = grlx-nom;
        };

        devShells.default = pkgs.mkShell {
          # Additional dev-shell environment variables can be set directly
          # MY_CUSTOM_DEVELOPMENT_VAR = "something else";

          # Extra inputs can be added here
          nativeBuildInputs = [
            commonArgs.buildInputs
            pkgs.cargo
            pkgs.rustc
            pkgs.rust-analyzer
          ];
        };
      };
    };
}

{
  description = "Dev environment for webstatus.dev";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    # Pinned to nixpkgs commit that provides golangci-lint 2.11.0
    nixpkgs-lint.url = "github:nixos/nixpkgs/0e6cdd5be64608ef630c2e41f8d51d484468492f";
  };

  outputs = { self, nixpkgs, flake-utils, nixpkgs-lint }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config.allowUnfree = true;
        };
        pkgs-lint = import nixpkgs-lint {
          inherit system;
        };
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            nodejs
            jdk25
            pkgs-lint.golangci-lint
            google-cloud-sdk
            kubectl
            minikube
            skaffold
            terraform
            tflint
            shellcheck
            gh
            jq
            netcat
            antlr4
            

          ];

          shellHook = ''
            export PS1="[nix-develop:\w]\$ "
            export MINIKUBE_PROFILE=webstatus-dev
            export DOCKER_BUILDKIT=1
            export ANTLR=antlr4
            export USE_DOCKER_BROWSER=true
            # Isolate Playwright browsers to this project
            export PLAYWRIGHT_BROWSERS_PATH="$PWD/.nix/browsers"
            
            echo "Entering Nix environment for webstatus.dev"
            
            # Print versions
            echo "Go version: $(go version)"
            echo "Node version: $(node --version)"
            echo "Java version: $(java --version | head -n 1)"
            echo "Terraform version: $(terraform version | head -n 1)"
            echo "TFLint version: $(tflint --version | head -n 1)"
            echo "Golangci-lint version: $(golangci-lint --version)"
            echo "Skaffold version: $(skaffold version)"
            echo "Minikube version: $(minikube version | head -n 1)"

            # Conditional Wayland support
            if [ -n "$WAYLAND_DISPLAY" ]; then
              echo "Wayland detected. Enabling Wayland support for browsers."
              export MOZ_ENABLE_WAYLAND=1
            else
              echo "X11 detected. Using standard display settings."
            fi

            # Playwright environment variables
            export PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS=1
          '';
        };
      });
}

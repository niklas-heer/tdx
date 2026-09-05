{
  description = "tdx - your todos, in markdown, done fast.";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs";
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        tdxContent = builtins.readFile ./tdx.toml;
        data = builtins.fromTOML tdxContent;
        pname = "tdx";
        version = data.version;
        description = data.description;
        allowDirty = true;
      in
      {
        packages.default = pkgs.buildGo127Module {
          inherit pname version;
          src =
            if allowDirty then
              pkgs.lib.cleanSourceWith {
                name = "${pname}";
                src = ./.;
                filter = name: type: pkgs.lib.cleanSourceFilter name type;
              }
            else
              ./.;
          vendorHash = "sha256-V7AF9AhUyekWMKHr0YRPPWPDtwksCrT5cJIJ3JLkLzo=";
          subPackages = [ "cmd/tdx" ];
          doCheck = false;
          ldflags = [
            "-X main.Version=${version}"
            "-X 'main.Description=${description}'"
          ];
        };
        apps.default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/tdx";
        };
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            pkg-config
            go_1_27
            gopls
            delve
            gotools
            go-tools
            golangci-lint
            git
            mise
          ];
          shellHook = ''echo "Run mise run check to validate tdx, or mise run dev to launch it."'';
        };
      }
    );
}

{
  description = "studio — the full life of a YouTube video, one binary";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      # Linux is the target (dev on x86_64, NixOS deploy); aarch64 for laptops.
      systems = [ "x86_64-linux" "aarch64-linux" ];
      forAllSystems = f:
        nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in {
      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          # go/gopls: build + language server.
          # ffmpeg-full: provides ffmpeg + ffprobe >= 6 (studio shells out; never
          #   links libav). rsync: archive. mpv: `search --play`.
          packages = [
            pkgs.go
            pkgs.gopls
            pkgs.ffmpeg-full
            pkgs.rsync
            pkgs.mpv
            pkgs.nodejs # runs the vendored SeaKim design conformance checker
          ];

          shellHook = ''
            echo "studio devShell — go $(go version | awk '{print $3}'), ffmpeg $(ffmpeg -version | head -1 | awk '{print $3}')"
          '';
        };
      });
    };
}

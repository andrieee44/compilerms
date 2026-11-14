{ lib, pkgs }:
pkgs.pkgsStatic.buildGoModule {
  name = "shlite";
  src = ./.;
  vendorHash = "sha256-3/tNPMYKxKjwprceoPRjRJuTNBwJSJE4LD8grnh+haI=";

  meta = {
    description = "shlite - shell-free wrapper with globbing and env filtering";
    homepage = "https://github.com/andrieee44/compilerms";
    license = lib.licenses.agpl3Plus;
    mainProgram = "shlite";
  };
}

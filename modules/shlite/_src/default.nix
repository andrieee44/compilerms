{ lib, pkgs }:
pkgs.pkgsStatic.buildGoModule {
  doCheck = true;
  name = "shlite";
  src = ./shlite;
  vendorHash = null;

  checkPhase = ''
    runHook preCheck
    go vet ./...
    runHook postCheck
  '';

  meta = {
    description = "shlite - ";
    homepage = "https://github.com/andrieee44/compilerms";
    license = lib.licenses.agpl3Plus;
    mainProgram = "shlite";
  };
}

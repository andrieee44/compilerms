{ lib, pkgs }:
pkgs.pkgsStatic.buildGoModule {
  doCheck = true;
  name = "sqlite3tmp";
  src = ./sqlite3tmp;
  vendorHash = "sha256-amXiSkx2PH7PtgL9RRYM8ueEQLxF9apzksGMEkJmBfQ=";

  checkPhase = ''
    runHook preCheck
    go vet ./...
    runHook postCheck
  '';

  meta = {
    description = "sqlite3tmp - ";
    homepage = "https://github.com/andrieee44/compilerms";
    license = lib.licenses.agpl3Plus;
    mainProgram = "sqlite3tmp";
  };
}

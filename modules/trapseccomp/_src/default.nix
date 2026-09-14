{ lib, pkgs }:
pkgs.pkgsStatic.buildGoModule {
  doCheck = true;
  name = "trapseccomp";
  src = ./trapseccomp;
  vendorHash = "sha256-st2b9YMP/SGKtKVAziwQ/5zlaoEt58lUE2OFbl9x6iY=";

  checkPhase = ''
    runHook preCheck
    go vet ./...
    runHook postCheck
  '';

  meta = {
    description = "trapseccomp - Seccomp violation trapping wrapper";
    homepage = "https://github.com/andrieee44/compilerms";
    license = lib.licenses.agpl3Plus;
    mainProgram = "trapseccomp";
  };
}

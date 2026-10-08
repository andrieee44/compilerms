{ lib, pkgs }:
pkgs.pkgsStatic.buildGoModule {
  doCheck = true;
  name = "compilerms";
  src = ./compilerms;
  vendorHash = "sha256-oAaNJjegZayVuNGy29BXX43gqrEkYU1AVT2LOrC6JpM=";

  checkPhase = ''
    runHook preCheck
    go vet ./...
    runHook postCheck
  '';

  meta = {
    description = "compilerms - Compiler Microservice";
    homepage = "https://github.com/andrieee44/compilerms";
    license = lib.licenses.agpl3Plus;
    mainProgram = "compilerms";
  };
}

{ lib, pkgs }:
pkgs.pkgsStatic.buildGoModule {
  doCheck = true;
  name = "compilerms";
  src = ./compilerms;
  vendorHash = "sha256-/PJf0Y6WIeQokQqac7t2JCeIjEhfC6NLZAgk2saYrSQ=";

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

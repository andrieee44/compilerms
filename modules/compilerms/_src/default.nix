{ lib, pkgs }:
pkgs.pkgsStatic.buildGoModule {
  name = "compilerms";
  src = ./.;
  vendorHash = "sha256-niL4i8kwPkMZUxKu0/Vl6GVYBChrdagPvSSVVa84Ukg=";

  meta = {
    description = "compilerms - Compiler Microservice";
    homepage = "https://github.com/andrieee44/compilerms";
    license = lib.licenses.agpl3Plus;
    mainProgram = "compilerms";
  };
}

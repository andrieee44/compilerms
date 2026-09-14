{
  perSystem =
    { pkgs, self', ... }:
    let
      compilerms = self'.packages.compilerms;
    in
    {
      checks.compilerms = compilerms;
      packages.compilerms = pkgs.callPackage ./_src { };

      apps.compilerms = {
        inherit (compilerms) meta;
        program = compilerms;
      };
    };
}

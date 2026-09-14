{
  perSystem =
    { pkgs, self', ... }:
    let
      trapseccomp = self'.packages.trapseccomp;
    in
    {
      checks.trapseccomp = trapseccomp;
      packages.trapseccomp = pkgs.callPackage ./_src { };

      apps.trapseccomp = {
        inherit (trapseccomp) meta;
        program = trapseccomp;
      };
    };
}

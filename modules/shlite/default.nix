{
  perSystem =
    { pkgs, self', ... }:
    let
      shlite = self'.packages.shlite;
    in
    {
      checks.shlite = shlite;
      packages.shlite = pkgs.callPackage ./_src { };

      apps.shlite = {
        inherit (shlite) meta;
        program = shlite;
      };
    };
}

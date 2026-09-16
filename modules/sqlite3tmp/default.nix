{
  perSystem =
    { pkgs, self', ... }:
    let
      sqlite3tmp = self'.packages.sqlite3tmp;
    in
    {
      checks.sqlite3tmp = sqlite3tmp;
      packages.sqlite3tmp = pkgs.callPackage ./_src { };

      apps.sqlite3tmp = {
        inherit (sqlite3tmp) meta;
        program = sqlite3tmp;
      };
    };
}

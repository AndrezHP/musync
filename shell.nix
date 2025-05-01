{
  pkgs ? import <nixpkgs> { },
}:
with pkgs;
pkgs.mkShell {
  packages = [
    gcc
    libcap
    godef
  ];
  buildInputs = [ go ];
  hardeningDisable = [ "fortify" ];
}

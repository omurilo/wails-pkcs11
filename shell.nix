{ pkgs ? import <nixpkgs> {} }:

let
  goVersion = "1_23";  # ou a versão que você quiser
in
pkgs.mkShell {
  name = "vault-unseal-dev";

  buildInputs = [
    pkgs."go_${goVersion}"
    pkgs.nodejs_20             # ou nodejs_18 se preferir
    pkgs.yarn                  # opcional, se usar yarn
    pkgs.softhsm
    pkgs.openssl
    pkgs.git
    pkgs.makeWrapper
    pkgs.pkg-config
    pkgs.libtool
    pkgs.autoconf
    pkgs.automake
    pkgs.gnumake
    pkgs.wget
    pkgs.cacert
    pkgs.opensc
  ];

  shellHook = ''
    echo "🛠️ Ambiente de desenvolvimento Wails + Go + SoftHSM2 iniciado"
    export GOPATH=$HOME/go
    export PATH=$GOPATH/bin:$PATH

    export PKCS11_LIB_PATH=${pkgs.softhsm}/lib/softhsm/libsofthsm2.so

    # Wails espera o libwebkit2gtk para builds GUI (se for rodar build GUI no Linux)
    export CGO_CFLAGS="$(pkg-config --cflags webkit2gtk-4.0)"
    export CGO_LDFLAGS="$(pkg-config --libs webkit2gtk-4.0)"

    # Caminho padrão do SoftHSM2
    export SOFTHSM2_CONF=$PWD/softhsm2.conf

    if ! command -v wails &> /dev/null; then
      echo "📦 Instalando Wails CLI (v2)..."
      go install github.com/wailsapp/wails/v2/cmd/wails@latest
    fi

    echo "✅ Execute 'wails doctor' para validar o setup"
  '';
}

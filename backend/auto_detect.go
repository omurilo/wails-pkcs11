package backend

import (
	"os"
	"runtime"
)

// FindPkcs11Module tenta encontrar a biblioteca PKCS#11 em locais comuns.
func FindPkcs11Module() string {
	var paths []string

	switch runtime.GOOS {
	case "linux":
		paths = []string{
			// SoftHSM
			"/usr/lib/x86_64-linux-gnu/softhsm/libsofthsm2.so",
			"/usr/lib/softhsm/libsofthsm2.so",
			// OpenSC
			"/usr/lib/x86_64-linux-gnu/opensc-pkcs11.so",
			// Yubico
			"/usr/lib/x86_64-linux-gnu/libykcs11.so",
		}
	case "darwin": // macOS
		paths = []string{
			// OpenSC (Homebrew)
			"/usr/local/lib/opensc-pkcs11.so",
			// Yubico
			"/usr/local/lib/libykcs11.dylib",
		}
	case "windows":
		system32 := os.Getenv("SystemRoot") + "\\System32\\"
		paths = []string{
			// OpenSC
			system32 + "opensc-pkcs11.dll",
			// Yubico
			os.Getenv("ProgramFiles") + "\\Yubico\\Yubico PIV Tool\\bin\\lib\\libykcs11.dll",
		}
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path // Retorna o primeiro que encontrar
		}
	}

	if env, ok := os.LookupEnv("PKCS11_LIB_PATH"); ok {
		return env
	}

	return "" // Não encontrou nenhum
}

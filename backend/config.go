package backend

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	AppName   = "VaultSealer"
	ConfigDir = ".vault-sealer"
	ConfigName = "config.json"
)

type Config struct {
	Pkcs11ModulePath string `json:"pkcs11_module_path"`
}

// getConfigPath retorna o caminho completo para o arquivo de configuração.
func getConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("não foi possível encontrar o diretório de configuração do usuário: %w", err)
	}
	appConfigDir := filepath.Join(configDir, ConfigDir)
	return filepath.Join(appConfigDir, ConfigName), nil
}

// LoadConfig carrega a configuração do arquivo JSON.
func LoadConfig() (Config, error) {
	var config Config
	path, err := getConfigPath()
	if err != nil {
		return config, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		// Se o arquivo não existe, é normal. Retorna config vazia.
		if os.IsNotExist(err) {
			return config, nil
		}
		return config, fmt.Errorf("falha ao ler arquivo de configuração: %w", err)
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, fmt.Errorf("falha ao decodificar config JSON: %w", err)
	}
	return config, nil
}

// SaveConfig salva a configuração no arquivo JSON.
func (c *Config) SaveConfig() error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}

	// Garante que o diretório de configuração exista
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("falha ao criar diretório de configuração: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("falha ao codificar config para JSON: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

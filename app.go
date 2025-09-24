package main

import (
	"ceremony-keys/backend"
	"context"
	"fmt"
	"runtime"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type Config backend.Config

// App struct
type App struct {
	ctx           context.Context
	config        backend.Config
	pkcs11Handler *backend.Pkcs11Handler
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Lógica de inicialização aprimorada
	config, err := backend.LoadConfig()
	if err != nil {
		// Loga o erro, mas continua, a app pode funcionar sem config
		fmt.Printf("AVISO: Falha ao carregar configuração: %v\n", err)
	}
	a.config = config

	// Se o caminho não está na config, tenta autodetectar
	if a.config.Pkcs11ModulePath == "" {
		fmt.Println("Nenhum caminho de módulo configurado, tentando autodetecção...")
		a.config.Pkcs11ModulePath = backend.FindPkcs11Module()
		if a.config.Pkcs11ModulePath != "" {
			fmt.Printf("Módulo detectado em: %s. Salvando configuração.\n", a.config.Pkcs11ModulePath)
			a.config.SaveConfig() // Salva o caminho detectado para o futuro
		}
	}

	// Inicializa o handler PKCS#11 se tivermos um caminho
	if a.config.Pkcs11ModulePath != "" {
		a.initializePkcs11Handler(a.config.Pkcs11ModulePath)
	} else {
		fmt.Println("Nenhum módulo PKCS#11 foi configurado ou detectado. A funcionalidade será limitada.")
	}
}

// Nova função helper para inicializar/reinicializar o handler
func (a *App) initializePkcs11Handler(modulePath string) {
	// Finaliza qualquer handler anterior
	if a.pkcs11Handler != nil {
		a.pkcs11Handler.Finalize()
	}

	a.pkcs11Handler = backend.NewPkcs11Handler(modulePath)
	if err := a.pkcs11Handler.Initialize(); err != nil {
		fmt.Printf("ERRO FATAL: Falha ao inicializar PKCS#11 com o caminho '%s': %v\n", modulePath, err)
		a.pkcs11Handler = nil // Garante que o handler é nulo em caso de erro
	} else {
		fmt.Printf("Módulo PKCS#11 inicializado com sucesso em '%s'\n", modulePath)
	}
}

func (a *App) shutdown(ctx context.Context) {
	if a.pkcs11Handler != nil {
		a.pkcs11Handler.Finalize()
	}
}

// --- NOVAS FUNÇÕES EXPOSTAS PARA CONFIGURAÇÃO ---

func (a *App) GetConfig() backend.Config {
	return a.config
}

func (a *App) SaveConfig(newConfig backend.Config) (string, error) {
	a.config = newConfig
	if err := a.config.SaveConfig(); err != nil {
		return "", err
	}

	// Reinicializa o handler com o novo caminho
	a.initializePkcs11Handler(newConfig.Pkcs11ModulePath)

	return "Configuração salva com sucesso!", nil
}

func (a *App) SelectModuleFile() (string, error) {
	pattern := "*.so"
	if runtime.GOOS == "windows" {
		pattern = "*.dll"
	} else if runtime.GOOS == "darwin" {
		pattern = "*.dylib,*.so"
	}

	selection, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Selecione a biblioteca PKCS#11",
		Filters: []wailsruntime.FileFilter{
			{
				DisplayName: "Bibliotecas PKCS#11",
				Pattern:     pattern,
			},
		},
	})
	if err != nil {
		return "", err
	}
	return selection, nil
}

func (a *App) SelectFileToShowDialog(title string, filterPattern string, filterDisplayName string) (string, error) {
	selection, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: title,
		Filters: []wailsruntime.FileFilter{
			{
				DisplayName: filterDisplayName,
				Pattern:     filterPattern,
			},
		},
	})
	if err != nil {
		return "", err
	}
	return selection, nil
}

// SelectSaveFileToShowDialog é a função wrapper para salvar arquivos.
func (a *App) SelectSaveFileToShowDialog(title string, defaultFilename string) (string, error) {
	selection, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: defaultFilename,
	})
	if err != nil {
		return "", err
	}
	return selection, nil
}

// GetSlots retorna a lista de tokens para o frontend.
// Wails converte o mapa Go para um objeto JavaScript automaticamente.
func (a *App) GetSlots() (map[uint]string, error) {
	if a.pkcs11Handler.Ctx == nil {
		return nil, fmt.Errorf("módulo PKCS#11 não foi inicializado corretamente")
	}
	return a.pkcs11Handler.GetSlotsWithInfo()
}

func (a *App) GetKeyLabelsForSlot(slotID uint, pin string) ([]string, error) {
	session, err := a.pkcs11Handler.OpenSession(slotID)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir sessão: %w", err)
	}
	defer a.pkcs11Handler.CloseSession(session)

	if err := a.pkcs11Handler.Login(session, pin); err != nil {
		// Retornamos o erro de login para o frontend tratar
		return nil, fmt.Errorf("falha no login (PIN incorreto?): %w", err)
	}
	defer a.pkcs11Handler.Logout(session)

	labels, err := a.pkcs11Handler.ListKeyLabels(session)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar labels das chaves: %w", err)
	}

	return labels, nil
}

// EncryptFile criptografa um arquivo usando o token.
func (a *App) EncryptFile(slotID uint, pin, keyLabel, inputFilePath, outputFilePath string) (string, error) {
	session, err := a.pkcs11Handler.OpenSession(slotID)
	if err != nil {
		return "", fmt.Errorf("falha ao abrir sessão: %w", err)
	}
	defer a.pkcs11Handler.CloseSession(session)

	if err := a.pkcs11Handler.Login(session, pin); err != nil {
		return "", fmt.Errorf("falha no login (PIN incorreto?): %w", err)
	}
	defer a.pkcs11Handler.Logout(session)

	_, pubKey, err := a.pkcs11Handler.FindKeyPair(session, keyLabel)
	if err != nil {
		return "", fmt.Errorf("falha ao encontrar par de chaves '%s': %w", keyLabel, err)
	}

	err = backend.HybridEncryptFile(a.pkcs11Handler.Ctx, session, pubKey, inputFilePath, outputFilePath)
	if err != nil {
		return "", fmt.Errorf("falha na criptografia: %w", err)
	}

	return fmt.Sprintf("Arquivo criptografado com sucesso em: %s", outputFilePath), nil
}

// DecryptFile descriptografa um arquivo e retorna seu conteúdo como string.
func (a *App) DecryptFile(slotID uint, pin, keyLabel, encryptedFilePath string) (string, error) {
	session, err := a.pkcs11Handler.OpenSession(slotID)
	if err != nil {
		return "", fmt.Errorf("falha ao abrir sessão: %w", err)
	}
	defer a.pkcs11Handler.CloseSession(session)

	if err := a.pkcs11Handler.Login(session, pin); err != nil {
		return "", fmt.Errorf("falha no login (PIN incorreto?): %w", err)
	}
	defer a.pkcs11Handler.Logout(session)

	privKey, _, err := a.pkcs11Handler.FindKeyPair(session, keyLabel)
	if err != nil {
		return "", fmt.Errorf("falha ao encontrar par de chaves '%s': %w", keyLabel, err)
	}

	decryptedBytes, err := backend.HybridDecryptFile(a.pkcs11Handler.Ctx, session, privKey, encryptedFilePath)
	if err != nil {
		return "", fmt.Errorf("falha na descriptografia: %w", err)
	}

	return string(decryptedBytes), nil
}

func (a *App) UnsealVault(decryptedKey string) {
	wailsruntime.LogInfo(a.ctx, fmt.Sprintf("The decrypted key is: %s", decryptedKey))
}

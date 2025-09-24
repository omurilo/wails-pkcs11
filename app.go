package main

import (
	"ceremony-keys/backend"
	"context"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx           context.Context
	pkcs11Handler *backend.Pkcs11Handler
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	
	// ATENÇÃO: ESTE CAMINHO DEVE SER CONFIGURÁVEL!
	// Substitua pelo caminho da sua biblioteca PKCS#11 (.so, .dll, .dylib)
	modulePath := os.Getenv("PKCS11_LIB_PATH")
	
	a.pkcs11Handler = backend.NewPkcs11Handler(modulePath)
	if err := a.pkcs11Handler.Initialize(); err != nil {
		// Em uma app real, você usaria o sistema de eventos do Wails
		// para notificar o frontend sobre a falha na inicialização.
		fmt.Printf("FATAL: Erro ao inicializar PKCS#11: %v\n", err)
	}
}

// shutdown is called when the app closes.
func (a *App) shutdown(ctx context.Context) {
	if a.pkcs11Handler != nil {
		a.pkcs11Handler.Finalize()
	}
}

func (a *App) SelectFileToShowDialog(title string, filterPattern string, filterDisplayName string) (string, error) {
	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
		Filters: []runtime.FileFilter{
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
	selection, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
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

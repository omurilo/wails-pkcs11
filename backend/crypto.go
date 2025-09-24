package backend

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/miekg/pkcs11"
)

// HybridEncryptFile implementa a criptografia híbrida.
func HybridEncryptFile(p *pkcs11.Ctx, session pkcs11.SessionHandle, pubKey pkcs11.ObjectHandle, inputPath, outputPath string) error {
	// 1. Gera chave simétrica (AES-256)
	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return fmt.Errorf("falha ao gerar chave AES: %w", err)
	}

	// 2. Criptografa a chave AES com a chave pública do token (RSA-PKCS)
	mechanism := []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_RSA_PKCS, nil)}
	if err := p.EncryptInit(session, mechanism, pubKey); err != nil {
		return fmt.Errorf("EncryptInit falhou: %w", err)
	}
	encryptedAesKey, err := p.Encrypt(session, aesKey)
	if err != nil {
		return fmt.Errorf("Encrypt falhou: %w", err)
	}

	// 3. Lê o arquivo de entrada
	plainText, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("falha ao ler arquivo de entrada: %w", err)
	}

	// 4. Criptografa o conteúdo do arquivo com a chave AES (usando AES-GCM)
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}
	cipherText := gcm.Seal(nonce, nonce, plainText, nil)

	// 5. Escreve o arquivo final no formato: [tamanho da chave AES criptografada (4 bytes)] + [chave AES criptografada] + [nonce+ciphertext]
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("falha ao criar arquivo de saída: %w", err)
	}
	defer outputFile.Close()

	keyLen := uint32(len(encryptedAesKey))
	if err := binary.Write(outputFile, binary.LittleEndian, keyLen); err != nil {
		return err
	}
	if _, err := outputFile.Write(encryptedAesKey); err != nil {
		return err
	}
	if _, err := outputFile.Write(cipherText); err != nil {
		return err
	}

	return nil
}

// HybridDecryptFile implementa a descriptografia.
func HybridDecryptFile(p *pkcs11.Ctx, session pkcs11.SessionHandle, privKey pkcs11.ObjectHandle, encryptedPath string) ([]byte, error) {
	encryptedFile, err := os.Open(encryptedPath)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir arquivo criptografado: %w", err)
	}
	defer encryptedFile.Close()

	// 1. Lê o tamanho da chave AES criptografada
	var keyLen uint32
	if err := binary.Read(encryptedFile, binary.LittleEndian, &keyLen); err != nil {
		return nil, fmt.Errorf("falha ao ler o tamanho da chave: %w", err)
	}

	// 2. Lê a chave AES criptografada
	encryptedAesKey := make([]byte, keyLen)
	if _, err := io.ReadFull(encryptedFile, encryptedAesKey); err != nil {
		return nil, fmt.Errorf("falha ao ler a chave criptografada: %w", err)
	}

	// 3. Descriptografa a chave AES usando a chave privada no token
	mechanism := []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_RSA_PKCS, nil)}
	if err := p.DecryptInit(session, mechanism, privKey); err != nil {
		return nil, fmt.Errorf("DecryptInit falhou: %w", err)
	}
	aesKey, err := p.Decrypt(session, encryptedAesKey)
	if err != nil {
		return nil, fmt.Errorf("Decrypt falhou (chave ou token incorreto?): %w", err)
	}

	// 4. Lê o resto do arquivo (nonce + ciphertext)
	cipherText, err := io.ReadAll(encryptedFile)
	if err != nil {
		return nil, err
	}

	// 5. Descriptografa os dados usando a chave AES
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(cipherText) < nonceSize {
		return nil, fmt.Errorf("ciphertext muito curto")
	}

	nonce, actualCipherText := cipherText[:nonceSize], cipherText[nonceSize:]
	plainText, err := gcm.Open(nil, nonce, actualCipherText, nil)
	if err != nil {
		return nil, fmt.Errorf("falha ao descriptografar dados (chave AES incorreta ou dados corrompidos): %w", err)
	}

	return plainText, nil
}

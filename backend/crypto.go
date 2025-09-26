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

func HybridEncryptFile(p *pkcs11.Ctx, session pkcs11.SessionHandle, pubKey pkcs11.ObjectHandle, inputPath, outputPath string) error {
	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return fmt.Errorf("falha ao gerar chave AES: %w", err)
	}

	mechanism := []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_RSA_PKCS, nil)}
	if err := p.EncryptInit(session, mechanism, pubKey); err != nil {
		return fmt.Errorf("EncryptInit falhou: %w", err)
	}
	encryptedAesKey, err := p.Encrypt(session, aesKey)
	if err != nil {
		return fmt.Errorf("Encrypt falhou: %w", err)
	}

	plainText, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("falha ao ler arquivo de entrada: %w", err)
	}

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

func HybridDecryptFile(p *pkcs11.Ctx, session pkcs11.SessionHandle, privKey pkcs11.ObjectHandle, encryptedPath string) ([]byte, error) {
	encryptedFile, err := os.Open(encryptedPath)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir arquivo criptografado: %w", err)
	}
	defer encryptedFile.Close()

	var keyLen uint32
	if err := binary.Read(encryptedFile, binary.LittleEndian, &keyLen); err != nil {
		return nil, fmt.Errorf("falha ao ler o tamanho da chave: %w", err)
	}

	encryptedAesKey := make([]byte, keyLen)
	if _, err := io.ReadFull(encryptedFile, encryptedAesKey); err != nil {
		return nil, fmt.Errorf("falha ao ler a chave criptografada: %w", err)
	}

	mechanism := []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_RSA_PKCS, nil)}
	if err := p.DecryptInit(session, mechanism, privKey); err != nil {
		return nil, fmt.Errorf("DecryptInit falhou: %w", err)
	}
	aesKey, err := p.Decrypt(session, encryptedAesKey)
	if err != nil {
		return nil, fmt.Errorf("Decrypt falhou (chave ou token incorreto?): %w", err)
	}

	cipherText, err := io.ReadAll(encryptedFile)
	if err != nil {
		return nil, err
	}

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

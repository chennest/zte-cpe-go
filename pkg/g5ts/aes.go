package g5ts

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

func decryptBase64(key [32]byte, b64 string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %w", err)
	}
	if len(data) < 12+16 {
		return "", fmt.Errorf("encrypted payload too short")
	}

	iv := data[:12]
	tag := data[12:28]
	ct := data[28:]

	combined := make([]byte, len(ct)+len(tag))
	copy(combined, ct)
	copy(combined[len(ct):], tag)

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCMWithNonceSize(block, 12)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := aesGCM.Open(nil, iv, combined, nil)
	if err != nil {
		return "", fmt.Errorf("AES-GCM decrypt failed: %w", err)
	}

	return string(plaintext), nil
}

func encryptBase64(key [32]byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCMWithNonceSize(block, 12)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	iv := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("failed to generate IV: %w", err)
	}

	ciphertext := aesGCM.Seal(nil, iv, []byte(plaintext), nil)

	ctLen := len(ciphertext) - 16
	tag := ciphertext[ctLen:]
	ct := ciphertext[:ctLen]

	ivHex := hex.EncodeToString(iv)
	tagHex := hex.EncodeToString(tag)
	ctHex := hex.EncodeToString(ct)

	combinedHex := ivHex + tagHex + ctHex
	combinedBytes := make([]byte, len(combinedHex)/2)
	for i := 0; i < len(combinedHex); i += 2 {
		b, _ := hex.DecodeString(combinedHex[i : i+2])
		combinedBytes[i/2] = b[0]
	}

	return base64.StdEncoding.EncodeToString(combinedBytes), nil
}

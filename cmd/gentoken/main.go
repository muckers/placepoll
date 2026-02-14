package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
)

func main() {
	// Load AES key from environment variable
	keyB64 := os.Getenv("AES_KEY")
	if keyB64 == "" {
		log.Fatal("AES_KEY environment variable is required")
	}

	aesKey, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		log.Fatalf("Failed to decode AES_KEY: %v", err)
	}

	if len(aesKey) != 32 {
		log.Fatalf("AES_KEY must be exactly 32 bytes, got %d bytes", len(aesKey))
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		panic(err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte("admin"), nil)
	token := base64.URLEncoding.EncodeToString(ciphertext)

	fmt.Println("Admin token:", token)
	fmt.Println("\nUse this token to access:")
	fmt.Println("  /links?t=" + token)
	fmt.Println("  /results?t=" + token)
}

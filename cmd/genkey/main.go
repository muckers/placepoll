package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
)

func main() {
	// Generate a random 32-byte key for AES-256
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		log.Fatal(err)
	}

	// Encode to base64 for easy storage in environment variables
	encoded := base64.StdEncoding.EncodeToString(key)

	fmt.Println("Generated AES-256 key (base64):")
	fmt.Println(encoded)
	fmt.Println("\nAdd this to your environment:")
	fmt.Println("export AES_KEY=\"" + encoded + "\"")
	fmt.Println("\nOr add to env.json:")
	fmt.Printf("\"AES_KEY\": \"%s\"\n", encoded)
}

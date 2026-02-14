package main

import (
	"encoding/base64"
	"log"
	"os"
)

// Destinations is the list of travel destinations to vote on
var Destinations = []string{
	"Chicago",
	"Milwaukee",
	"St. Louis",
	"Omaha",
	"Louisville",
	"Memphis",
	"Austin",
	"San Antonio",
	"Iowa City",
	"Des Moines",
	"Denver",
	"Madison",
	"Springfield",
	"Cedar Rapids",
}

// Voters is the list of authorized voters
var Voters = []string{
	"Lesley",
	"Casey",
	"James",
	"Rebecca",
	"Kate",
	"Monica",
	"Aaron",
}

// AdminUsers can access /results and /links
var AdminUsers = []string{
	"admin",
}

// AESKey is the 32-byte key for AES-256-GCM encryption loaded from environment
var AESKey []byte

func init() {
	// Load AES key from environment variable
	keyB64 := os.Getenv("AES_KEY")
	if keyB64 == "" {
		log.Fatal("AES_KEY environment variable is required")
	}

	var err error
	AESKey, err = base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		log.Fatalf("Failed to decode AES_KEY: %v", err)
	}

	if len(AESKey) != 32 {
		log.Fatalf("AES_KEY must be exactly 32 bytes, got %d bytes", len(AESKey))
	}
}

package jwtdecode

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var (
	publicKeyCache     = make(map[string]*rsa.PublicKey)
	publicKeyCacheTime = make(map[string]time.Time)
	publicKeyCacheLock = sync.RWMutex{}
	cacheTTL           = 30 * time.Minute
)

// Fetch and find the public key from the provided URL
func FetchPublicKeyFromURL(url string, token *jwt.Token, forceRefresh bool) (*rsa.PublicKey, error) {
	// Extract kid from the token header
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("missing kid in token header")
	}

	// Check if the key is cached and if it's valid
	publicKeyCacheLock.RLock()
	key, exists := publicKeyCache[kid]
	cacheTime, cacheExists := publicKeyCacheTime[kid]
	publicKeyCacheLock.RUnlock()

	if exists && cacheExists && time.Since(cacheTime) <= cacheTTL && !forceRefresh {
		return key, nil
	}

	log.Println("Refreshing key.")

	// Fetch the public key set from the URL
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch public keys: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch public keys: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key response: %v", err)
	}

	// Parse the JSON response
	var keySet struct {
		Keys []struct {
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}

	if err := json.Unmarshal(body, &keySet); err != nil {
		return nil, fmt.Errorf("failed to parse public key JSON: %v", err)
	}

	// Update the cache for all keys in the response
	publicKeyCacheLock.Lock()
	defer publicKeyCacheLock.Unlock()

	for _, keyData := range keySet.Keys {
		nBytes, err := decodeBase64URL(keyData.N)
		if err != nil {
			return nil, fmt.Errorf("failed to decode public key modulus: %v", err)
		}

		eBytes, err := decodeBase64URL(keyData.E)
		if err != nil {
			return nil, fmt.Errorf("failed to decode public key exponent: %v", err)
		}

		n := new(big.Int).SetBytes(nBytes)
		e := int(bigEndianBytesToInt(eBytes))
		publicKeyCache[keyData.Kid] = &rsa.PublicKey{
			N: n,
			E: e,
		}
		publicKeyCacheTime[keyData.Kid] = time.Now()
	}

	// Return the requested key
	if key, ok := publicKeyCache[kid]; ok {
		return key, nil
	}

	return nil, errors.New("no matching public key found")
}

// Helper to decode Base64URL strings
func decodeBase64URL(value string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(value)
}

// Helper to convert big-endian bytes to int
func bigEndianBytesToInt(b []byte) int {
	result := 0
	for _, v := range b {
		result = result<<8 + int(v)
	}
	return result
}

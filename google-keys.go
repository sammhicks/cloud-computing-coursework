package main

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
)

// Key is a JWK key
type Key struct {
	Kty string
	Alg string
	Use string
	Kid string
	N   string
	E   string
}

type jwkKeys struct {
	Keys []Key
}

// GoogleKeys loads the Google Keys
func GoogleKeys() (keys []Key, err error) {
	r, err := http.Get("https://www.googleapis.com/oauth2/v3/certs")

	if err != nil {
		return
	}

	defer r.Body.Close()

	var rawKeys jwkKeys

	decoder := json.NewDecoder(r.Body)

	err = decoder.Decode(&rawKeys)

	if err != nil {
		return
	}

	keys = rawKeys.Keys
	return
}

// LookupRSAKey searches through a list and finds an RSA key with the correct id, algorithm, and string
func LookupRSAKey(keys []Key, id string, algorithm string, use string) (*rsa.PublicKey, error) {
	for _, key := range keys {
		if key.Kty == "RSA" && key.Alg == algorithm && key.Use == use {
			nBytes, err := base64.RawURLEncoding.DecodeString(key.N)

			if err != nil {
				return nil, err
			}

			eBytes, err := base64.RawURLEncoding.DecodeString(key.E)

			if err != nil {
				return nil, err
			}

			var n, e big.Int

			n.SetBytes(nBytes)
			e.SetBytes(eBytes)

			return &rsa.PublicKey{
				N: &n,
				E: int(e.Int64())}, nil
		}
	}

	return nil, errors.New("Key not found")
}

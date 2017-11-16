package main

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type tokenHeader struct {
	Alg string
	Kid string
}

type tokenBody struct {
	Aud   string
	Email string
	Exp   int64
	IAt   int64
	Iss   string
	Sub   string
}

// VerifyToken verified the given token with the given public keys and app unique id
func VerifyToken(token string, keys []Key, aud string) (id string, email string, err error) {
	sections := strings.Split(token, ".")

	if len(sections) != 3 {
		err = errors.New("token does not contain header, body, and signiature")
		return
	}

	hBytes, err := base64.RawURLEncoding.DecodeString(sections[0])

	if err != nil {
		return
	}

	var header tokenHeader

	err = json.Unmarshal(hBytes, &header)

	if err != nil {
		return
	}

	key, err := LookupRSAKey(keys, header.Kid, header.Alg, "sig")

	if err != nil {
		return
	}

	sBytes, err := base64.RawURLEncoding.DecodeString(sections[2])

	if err != nil {
		return
	}

	switch header.Alg {
	case "RS256":
		hashed := sha256.Sum256([]byte(sections[0] + "." + sections[1]))

		err = rsa.VerifyPKCS1v15(key, crypto.SHA256, hashed[:], sBytes)
		if err != nil {
			return
		}
	default:
		err = errors.New("algorithm not supported")
		return
	}

	bBytes, err := base64.RawURLEncoding.DecodeString(sections[1])

	if err != nil {
		return
	}

	var body tokenBody

	err = json.Unmarshal(bBytes, &body)

	if err != nil {
		return
	}

	if body.Aud != aud {
		err = errors.New("aud wrong")
		return
	}

	if body.Iss != "accounts.google.com" && body.Iss != "https://accounts.google.com" {
		err = errors.New("iss wrong")
		return
	}

	if time.Now().Before(time.Unix(body.IAt, 0)) {
		err = errors.New("issued in the past")
		return
	}

	if time.Now().After(time.Unix(body.Exp, 0)) {
		err = errors.New("expired")
		return
	}

	id = body.Sub

	email = body.Email

	return
}

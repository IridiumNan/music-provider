package main

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func getChallenge(baseURL string) ([]byte, error) {
	resp, err := http.Get(baseURL + "/auth/challenge")
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func signNonce(priv *rsa.PrivateKey, nonce []byte) ([]byte, error) {
	digest := sha256.Sum256(nonce)

	return rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, digest[:])
}

func getList(baseURL string, nonce []byte, signature []byte) {
	var body bytes.Buffer
	nonceEncoded := make([]byte, base64.RawStdEncoding.EncodedLen(len(nonce)))
	signatureEncoded := make([]byte, base64.RawStdEncoding.EncodedLen(len(signature)))

	base64.RawStdEncoding.Encode(nonceEncoded, nonce)
	base64.RawStdEncoding.Encode(signatureEncoded, signature)

	// body.Write(nonceEncoded)
	// body.WriteRune('\n')
	body.Write(signatureEncoded)

	resp, err := http.Post(baseURL+"/list/text", "plain/text", &body)
	if err != nil {
		log.Fatal("error while request for music list", err)
	}

	defer resp.Body.Close()

	io.Copy(os.Stdout, resp.Body)
}

func loadPrivKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error while reading file: %s", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	keyAny, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKCS8: %w", err)
	}

	priv, ok := keyAny.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA private key")
	}

	return priv, nil
}

func main() {
	baseURL := "http://localhost:8080"
	key, err := loadPrivKey("keys/private.pem")
	if err != nil {
		log.Fatal("while loading private key", err)
	}

	nonce, err := getChallenge(baseURL)
	if err != nil {
		log.Fatal("while sending request for challenge", err)
	}

	sig, err := signNonce(key, nonce)
	if err != nil {
		log.Fatal("while signing by private key", err)
	}

	getList(baseURL, nonce, sig)
}

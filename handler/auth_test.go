package handler_test

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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mp3-stream/handler"

	_ "embed"
)

//go:embed test_key/public.pem
var pubKeyData []byte

//go:embed test_key/private.pem
var privKeyData []byte

func loadPrivKey(data []byte) (*rsa.PrivateKey, error) {
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

func successHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func newAuthReq(nonce []byte, privKey *rsa.PrivateKey) (*http.Request, error) {
	// got digital signature by private key and nonce
	digest := sha256.Sum256(nonce)
	sig, err := rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, digest[:])
	if err != nil {
		return nil, fmt.Errorf("while signing with private key, err: %s", err)
	}

	var body bytes.Buffer
	signatureEncoded := make([]byte, base64.StdEncoding.EncodedLen(len(sig)))

	base64.StdEncoding.Encode(signatureEncoded, sig)

	body.Write(nonce)
	body.Write([]byte("\n"))
	body.Write(signatureEncoded)

	return httptest.NewRequest(http.MethodPost, "/", &body), nil
}

func TestAuthBasic(t *testing.T) {
	privKey, err := loadPrivKey(privKeyData)
	if err != nil {
		t.Errorf("error while loading private key: %s", err)
	}

	auth, err := handler.NewAuth(pubKeyData)
	if err != nil {
		t.Errorf("while creating a new auth obj with public key, err: %s", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	// request for challenge first
	auth.HandleChallenge(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("error while getting challenge string, err: %s", err)
	}

	// read challenge nonce then sign with private key

	nonce, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	req, err = newAuthReq(nonce, privKey)
	if err != nil {
		t.Error(err)
	}

	testHandler := auth.MiddleWare(successHandler)

	w = httptest.NewRecorder()
	testHandler(w, req)
	resp = w.Result()

	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected auth success, got failed")
	}

	// send request again, it should be failed as this challenge has been used (expired)
	req, err = newAuthReq(nonce, privKey)
	if err != nil {
		t.Error(err)
	}

	w = httptest.NewRecorder()
	testHandler(w, req)
	resp = w.Result()

	if resp.StatusCode == http.StatusOK {
		t.Errorf("expired key used !!!, error")
	}
}

func TestAuthTimeout(t *testing.T) {
	privKey, err := loadPrivKey(privKeyData)
	if err != nil {
		t.Errorf("error while loading private key: %s", err)
	}

	auth, err := handler.NewAuthWitTimeout(pubKeyData, 100*time.Second, 3*time.Second)
	if err != nil {
		t.Errorf("while creating a new auth obj with public key, err: %s", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	// request for challenge first
	auth.HandleChallenge(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("error while getting challenge string, err: %s", err)
	}

	// read challenge nonce then sign with private key

	nonce, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	req, err = newAuthReq(nonce, privKey)
	if err != nil {
		t.Error(err)
	}

	testHandler := auth.MiddleWare(successHandler)

	// should be success
	time.Sleep(1 * time.Second)
	w = httptest.NewRecorder()
	testHandler(w, req)
	resp = w.Result()
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expect no timeout and auth success, got failed")
	}

	// create a new request for timeout test
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	w = httptest.NewRecorder()

	// request for challenge first
	auth.HandleChallenge(w, req)

	resp = w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("error while getting challenge string, err: %s", err)
	}

	// read challenge nonce then sign with private key

	nonce, _ = io.ReadAll(resp.Body)
	resp.Body.Close()

	req, err = newAuthReq(nonce, privKey)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(5 * time.Second)

	w = httptest.NewRecorder()
	testHandler(w, req)
	resp = w.Result()
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Errorf("challenge should be timeout, got success response")
	}
}

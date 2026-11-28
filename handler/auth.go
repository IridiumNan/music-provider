package handler

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
	"log/slog"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type hashVal struct {
	// hash algo is sha256 (32 bytes)
	hashByte [32]byte
	expired  atomic.Bool
	timeout  time.Duration
}

// verify based on the signature
// if success return nil
// else return failed reason with error wrapper
func (hv *hashVal) verify(pub *rsa.PublicKey, sig []byte) bool {
	if hv.expired.Load() {
		return false
	}

	hv.expired.Store(true)
	err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, hv.hashByte[:], sig)

	return err == nil
}

// start begin a new timer for hash expire
// you should run with gorutine
func (hv *hashVal) start() {
	<-time.After(hv.timeout)
	hv.expired.Store(true)
}

// newHash with default timeout 10 seconds
func newHash(hash [32]byte) *hashVal {
	return newHashWithTimeout(hash, 15*time.Second)
}

func newHashWithTimeout(hash [32]byte, timeout time.Duration) *hashVal {
	val := hashVal{
		hashByte: hash,
		expired:  atomic.Bool{},
		timeout:  timeout,
	}
	go val.start()
	return &val
}

type Auth struct {
	pubKey  *rsa.PublicKey
	hashSet map[string]*hashVal
	mu      sync.RWMutex

	expireTiemout time.Duration
}

func (a *Auth) HandleChallenge(w http.ResponseWriter, r *http.Request) {
	chBytes := []byte(rand.Text())
	_, err := w.Write(chBytes)
	if err != nil {
		slog.Error("error when sending new challenge to client", "err", err)
	}

	a.mu.Lock()
	a.hashSet[string(chBytes)] = newHashWithTimeout(sha256.Sum256(chBytes), a.expireTiemout)
	a.mu.Unlock()
}

// clean all expired hash on hashSet
func (a *Auth) autoClean(interval time.Duration) {
	for {
		<-time.After(interval)
		a.mu.Lock()
		startLen := len(a.hashSet)
		newSet := make(map[string]*hashVal, startLen)
		for nonce, h := range a.hashSet {
			if !h.expired.Load() {
				newSet[nonce] = h
			}
		}

		a.hashSet = newSet
		// endLen := len(newSet)
		// slog.Debug("finish clear expired set", "original_len", startLen, "new_len", endLen)
		a.mu.Unlock()
	}
}

func (a *Auth) Verify(nonce, sig []byte) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	h, ok := a.hashSet[string(nonce)]
	if !ok {
		return false
	}

	return h.verify(a.pubKey, sig)
}

func (a *Auth) MiddleWare(next func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		data, err := io.ReadAll(r.Body)
		nonce, receiveBase64, f := bytes.Cut(data, []byte("\n"))
		if !f {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "invalid format")
			return
		}
		if err != nil {
			slog.Error("error while reading signature", "err", err)
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "error while reading signature")
			return
		}

		receiveSig := make([]byte, base64.RawStdEncoding.DecodedLen(len(receiveBase64)))
		base64.RawStdEncoding.Decode(receiveSig, receiveBase64)
		if !a.Verify(nonce, receiveSig) {
			slog.Error("Failed to verify signature", "remote_addr", r.RemoteAddr)
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "Failed to verify")
			return
		}

		next(w, r)
	}
}

func NewAuth(publicKeyData []byte) (*Auth, error) {
	return NewAuthWitTimeout(publicKeyData, 30*time.Second, 15*time.Second)
}

func NewAuthWitTimeout(publicKeyData []byte, cleanInterval time.Duration, challengeTimeout time.Duration) (*Auth, error) {
	block, _ := pem.Decode(publicKeyData)
	pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		slog.Error("while parsing public key", "error", err)
		os.Exit(1)
	}
	pub := pubAny.(*rsa.PublicKey)

	au := Auth{
		pubKey:        pub,
		hashSet:       make(map[string]*hashVal, 10),
		mu:            sync.RWMutex{},
		expireTiemout: challengeTimeout,
	}
	go au.autoClean(cleanInterval)

	return &au, nil
}

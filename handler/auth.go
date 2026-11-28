package handler

import (
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
	hashByte      [32]byte
	expired       atomic.Bool
	timeoutSecond int
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
	<-time.After(time.Duration(hv.timeoutSecond) * time.Second)
	hv.expired.Store(true)
}

// newHash with default timeout 10 seconds
func newHash(hash [32]byte) *hashVal {
	return newHashWithTimeout(hash, 10)
}

func newHashWithTimeout(hash [32]byte, timeoutSecond int) *hashVal {
	val := hashVal{
		hashByte:      hash,
		expired:       atomic.Bool{},
		timeoutSecond: timeoutSecond,
	}
	go val.start()
	return &val
}

type Auth struct {
	pubKey  *rsa.PublicKey
	hashSet []*hashVal
	mu      sync.RWMutex
}

func (a *Auth) HandleChallenge(w http.ResponseWriter, r *http.Request) {
	chBytes := []byte(rand.Text())
	_, err := w.Write(chBytes)
	if err != nil {
		slog.Error("error when sending new challenge to client", "err", err)
	}

	a.mu.Lock()
	a.hashSet = append(a.hashSet, newHash(sha256.Sum256(chBytes)))
	a.mu.Unlock()
}

// clean all expired hash on hashSet
func (a *Auth) autoClean(interval time.Duration) {
	for {
		<-time.After(interval)
		a.mu.Lock()
		startLen := len(a.hashSet)
		newSet := make([]*hashVal, 0, startLen)
		for _, h := range a.hashSet {
			if !h.expired.Load() {
				newSet = append(newSet, h)
			}
		}

		a.hashSet = newSet
		endLen := len(newSet)
		slog.Debug("finish clear expired set", "original_len", startLen, "new_len", endLen)
		a.mu.Unlock()
	}
}

func (a *Auth) Verify(sig []byte) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, h := range a.hashSet {
		if h.verify(a.pubKey, sig) {
			return true
		}
	}

	return false
}

func (a *Auth) MiddleWare(next func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		receiveBase64, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Error("error while reading signature", "err", err)
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "error while reading signature")
			return
		}

		receiveSig := make([]byte, base64.RawStdEncoding.DecodedLen(len(receiveBase64)))
		base64.RawStdEncoding.Decode(receiveSig, receiveBase64)
		if !a.Verify(receiveSig) {
			slog.Error("Failed to verify signature", "remote_addr", r.RemoteAddr)
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "Failed to verify")
			return
		}

		next(w, r)
	}
}

func NewAuth(publicKeyData []byte) (*Auth, error) {
	block, _ := pem.Decode(publicKeyData)
	pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		slog.Error("while parsing public key", "error", err)
		os.Exit(1)
	}
	pub := pubAny.(*rsa.PublicKey)

	au := Auth{
		pubKey:  pub,
		hashSet: make([]*hashVal, 0, 10),
		mu:      sync.RWMutex{},
	}
	go au.autoClean(30 * time.Second)

	return &au, nil
}

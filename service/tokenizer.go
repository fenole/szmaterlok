package service

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"filippo.io/age"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// SessionSimpleTokenizer is a simple key/value storage for
// string tokens and session state of users.
type SessionSimpleTokenizer struct {
	gen     IDGenerator
	storage map[string]SessionState
	mtx     *sync.RWMutex
	base64  *base64.Encoding
}

// NewSessionSimpleTokenizer is default and safe constructor for SessionSimpleTokenizer.
func NewSessionSimpleTokenizer() *SessionSimpleTokenizer {
	return &SessionSimpleTokenizer{
		gen:     IDGeneratorFunc(uuid.NewString),
		storage: make(map[string]SessionState),
		mtx:     &sync.RWMutex{},
		base64:  base64.URLEncoding,
	}
}

// TokenEncode returns tokenized string which represents session state and
// can be decoded with the same interface implementation.
func (t *SessionSimpleTokenizer) TokenEncode(state SessionState) (string, error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	token := t.gen.GenerateID()

	hostname, err := os.Hostname()
	if err == nil {
		token = hostname + "/" + token
	}

	t.storage[token] = state

	return t.base64.EncodeToString([]byte(token)), nil
}

var ErrMissingSessionToken = errors.New("session: missing token")

// TokenDecode decodes given string token into valid session state.
func (t *SessionSimpleTokenizer) TokenDecode(token string) (*SessionState, error) {
	b, err := t.base64.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode token: %w", err)
	}

	t.mtx.RLock()
	defer t.mtx.RUnlock()

	token = string(b)
	s, ok := t.storage[token]
	if !ok {
		return nil, ErrMissingSessionToken
	}

	return &s, nil
}

// SessionAgeTokenizer encodes and decodes session state token.
type SessionAgeTokenizer struct {
	recipient age.Recipient
	identity  age.Identity
	base64    *base64.Encoding
}

// NewSessionAgeTokenizer returns SessionAgeTokenizer which encrypts
// and decrypts tokens with given secret. Make sure secret is
// long enough and has high entropy.
func NewSessionAgeTokenizer(secret string) (*SessionAgeTokenizer, error) {
	r, err := age.NewScryptRecipient(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create new scrypt recipient: %w", err)
	}

	i, err := age.NewScryptIdentity(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create new scrypt identity: %w", err)
	}

	return &SessionAgeTokenizer{
		recipient: r,
		identity:  i,
		base64:    base64.URLEncoding,
	}, nil
}

// TokenEncode encodes given session state into encrypted and base64 encoded
// token string, which can be used to safely store session state in users
// browser.
func (st *SessionAgeTokenizer) TokenEncode(state SessionState) (string, error) {
	buff := &bytes.Buffer{}

	wc, err := age.Encrypt(buff, st.recipient)
	if err != nil {
		return "", fmt.Errorf("failed to create encrypted writer: %w", err)
	}

	if err := json.NewEncoder(wc).Encode(state); err != nil {
		return "", fmt.Errorf("failed to encode state into json: %w", err)
	}

	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("failed to encrypt session state: %w", err)
	}

	return st.base64.EncodeToString(buff.Bytes()), nil
}

// TokenDecode decodes given base64 encoded and encrypted token into
// SessionState.
func (st *SessionAgeTokenizer) TokenDecode(token string) (*SessionState, error) {
	b, err := st.base64.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token from base64: %w", err)
	}

	buff := bytes.NewBuffer(b)

	src, err := age.Decrypt(buff, st.identity)
	if err != nil {
		return nil, fmt.Errorf("failed to created encrypted reader: %w", err)
	}

	out := &bytes.Buffer{}

	if _, err := io.Copy(out, src); err != nil {
		return nil, fmt.Errorf("failed to read encrypted token: %w", err)
	}

	res := &SessionState{}
	if err := json.NewDecoder(out).Decode(res); err != nil {
		return nil, fmt.Errorf("failed to decode session state to json: %w", err)
	}

	return res, nil
}

// SessionAESTokenizer implements stateless SessionTokenizer interface
// with AES/CFB encryption.
type SessionAESTokenizer struct {
	block  cipher.Block
	base64 *base64.Encoding
}

// NewSessionAESTokenizer returns AES session tokenizer. It is only safe
// constructor and the default one also.
//
// The secret argument should be the AES key, either 16, 24, or 32 bytes
// to select AES-128, AES-192, or AES-256.
func NewSessionAESTokenizer(secret []byte) (*SessionAESTokenizer, error) {
	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, fmt.Errorf("Failed to crete new AES cipher block: %w", err)
	}

	return &SessionAESTokenizer{
		block:  block,
		base64: base64.URLEncoding,
	}, nil
}

func (st *SessionAESTokenizer) newIV() ([]byte, error) {
	iv := make([]byte, st.block.BlockSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("failed to fill iv: %w", err)
	}

	return iv, nil
}

// TokenEncode returns tokenized string which represents session state and
// can be decoded with the same interface implementation.
func (st *SessionAESTokenizer) TokenEncode(state SessionState) (string, error) {
	b, err := json.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("failed to encode state into json: %w", err)
	}

	iv, err := st.newIV()
	if err != nil {
		return "", fmt.Errorf("failed to encode state with IV: %w", err)
	}

	cfb := cipher.NewCFBEncrypter(st.block, iv)
	res := make([]byte, len(b))
	cfb.XORKeyStream(res, b)
	return st.base64.EncodeToString(iv) + ":" + st.base64.EncodeToString(res), nil
}

// TokenDecode decodes given string token into valid session state.
func (st *SessionAESTokenizer) TokenDecode(token string) (*SessionState, error) {
	splitted := strings.Split(token, ":")

	iv, err := st.base64.DecodeString(splitted[0])
	if err != nil {
		return nil, fmt.Errorf("failed to decode iv from base64: %w", err)
	}

	b, err := st.base64.DecodeString(splitted[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode token from base64: %w", err)
	}

	res := &SessionState{}
	cfb := cipher.NewCFBDecrypter(st.block, iv)
	decrypted := make([]byte, len(b))
	cfb.XORKeyStream(decrypted, b)

	if err := json.Unmarshal(decrypted, res); err != nil {
		return nil, fmt.Errorf("failed to decode json state: %w", err)
	}

	return res, nil
}

type sessionTokenizerCacheEntry struct {
	value SessionState
	timer *time.Timer
}

// SessionTokenizerCache wraps SessionTokenizer interface and extends it
// with concurrent-safe in-memory cache storage.
type SessionTokenizerCache struct {
	wrapped SessionTokenizer
	timeout time.Duration
	log     *logrus.Logger
	mtx     *sync.RWMutex
	cache   map[string]sessionTokenizerCacheEntry
}

// SessionTokenizerCacheBuilder holds build arguments for SessionTokenizerCache.
type SessionTokenizerCacheBuilder struct {
	Wrapped SessionTokenizer
	Timeout time.Duration
	Logger  *logrus.Logger
}

// NewSessionTokenizerCache is default and safe constructor for SessionTokenizerCache.
func NewSessionTokenizerCache(b SessionTokenizerCacheBuilder) *SessionTokenizerCache {
	return &SessionTokenizerCache{
		wrapped: b.Wrapped,
		timeout: b.Timeout,
		log:     b.Logger,
		mtx:     &sync.RWMutex{},
		cache:   make(map[string]sessionTokenizerCacheEntry),
	}
}

// TokenEncode returns tokenized string which represents session state and
// can be decoded with the same interface implementation.
func (c *SessionTokenizerCache) TokenEncode(state SessionState) (string, error) {
	return c.wrapped.TokenEncode(state)
}

// TokenDecode decodes given string token into valid session state.
func (c *SessionTokenizerCache) TokenDecode(token string) (*SessionState, error) {
	// Read cache entry from map. Wrap it with read lock.
	c.mtx.RLock()
	entry, ok := c.cache[token]
	c.mtx.RUnlock()
	if !ok {
		// There is no entry in cache. Decode token manually.
		res, err := c.wrapped.TokenDecode(token)
		if err != nil {
			return nil, err
		}

		// Begin write transaction.
		c.mtx.Lock()

		// Add new cache entry for given token.
		c.cache[token] = sessionTokenizerCacheEntry{
			value: *res,

			// Fire garbage collection for given token after cache timeout.
			timer: time.AfterFunc(c.timeout, func() {
				s := *res
				c.mtx.Lock()
				defer c.mtx.Unlock()
				delete(c.cache, token)
				c.log.WithFields(logrus.Fields{
					"userID":   s.ID,
					"nickname": s.Nickname,
				}).Debug("Garbage collection of tokenizer cache.")
			}),
		}

		// End write transaction.
		c.mtx.Unlock()

		return res, nil
	}

	entry.timer.Reset(c.timeout)
	return &entry.value, nil
}

// SessionTokenizerFactory initiates tokenizer for szmaterlok based
// on configuration variables.
type SessionTokenizerFactory struct {
	Timeout time.Duration
	Logger  *logrus.Logger
}

var ErrInvalidTokenizerType = errors.New("session: invalid tokenizer type name")

// Tokenizer builds session tokenizer wrapped with cache based on
// the environmental variable from configuration.
func (f *SessionTokenizerFactory) Tokenizer(config *ConfigVariables) (SessionTokenizer, error) {
	cacheBuilder := SessionTokenizerCacheBuilder{
		Wrapped: nil,
		Timeout: f.Timeout,
		Logger:  f.Logger,
	}

	switch config.Tokenizer {

	case ConfigTokenizerSimple:
		f.Logger.Info("Chose simple tokenizer backend.")
		t := NewSessionSimpleTokenizer()
		cacheBuilder.Wrapped = t
		return NewSessionTokenizerCache(cacheBuilder), nil

	case ConfigTokenizerAge:
		f.Logger.Info("Chose age tokenizer backend.")
		t, err := NewSessionAgeTokenizer(config.SessionSecret)
		if err != nil {
			return nil, err
		}
		cacheBuilder.Wrapped = t
		return NewSessionTokenizerCache(cacheBuilder), nil

	case ConfigTokenizerAES:
		f.Logger.Info("Chose AES tokenizer backend.")
		t, err := NewSessionAESTokenizer([]byte(config.SessionSecret))
		if err != nil {
			return nil, err
		}
		cacheBuilder.Wrapped = t
		return NewSessionTokenizerCache(cacheBuilder), nil

	default:
		return nil, ErrInvalidTokenizerType
	}
}

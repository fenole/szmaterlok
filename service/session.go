package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"filippo.io/age"
)

// SessionState is model for user sessions stored in
// browser or any other storage.
type SessionState struct {
	Nickname  string    `json:"nck"`
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"cat"`
	ExpireAt  time.Time `json:"eat"`
}

// SessionStateFactory creates new unique session states.
type SessionStateFactory struct {
	ExpirationTime time.Duration
	IDGenerator
	Clock
}

// MakeState creates new unique session state for given nickname.
func (ssf SessionStateFactory) MakeState(nickname string) SessionState {
	now := ssf.Now()
	return SessionState{
		Nickname:  nickname,
		ID:        ssf.GenerateID(),
		CreatedAt: now,
		ExpireAt:  now.Add(ssf.ExpirationTime),
	}
}

// SessionTokenizer encodes and decodes session state token.
type SessionTokenizer struct {
	recipient age.Recipient
	identity  age.Identity
}

// NewSessionTokenizer returns SessionTokenizer which encrypts
// and decrypts tokens with given secret. Make sure secret is
// long enough and has high entropy.
func NewSessionTokenizer(secret string) (*SessionTokenizer, error) {
	r, err := age.NewScryptRecipient(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create new scrypt recipient: %w", err)
	}

	i, err := age.NewScryptIdentity(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create new scrypt identity: %w", err)
	}

	return &SessionTokenizer{
		recipient: r,
		identity:  i,
	}, nil
}

// TokenEncode encodes given session state into encrypted and base64 encoded
// token string, which can be used to safely store session state in users
// browser.
func (st *SessionTokenizer) TokenEncode(state SessionState) (string, error) {
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

	return base64.StdEncoding.EncodeToString(buff.Bytes()), nil
}

// TokenDecode decodes given base64 encoded and encrypted token into
// SessionState.
func (st *SessionTokenizer) TokenDecode(token string) (*SessionState, error) {
	b, err := base64.StdEncoding.DecodeString(token)
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

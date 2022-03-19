package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"filippo.io/age"
	"github.com/google/uuid"
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

// DefaultSessionStateFactory is default constructor for SessionStateFactory.
func DefaultSessionStateFactory() *SessionStateFactory {
	return &SessionStateFactory{
		// Default session state is valid for one week.
		ExpirationTime: time.Hour * 24 * 7,
		IDGenerator:    IDGeneratorFunc(uuid.NewString),
		Clock:          ClockFunc(time.Now),
	}
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
	base64    *base64.Encoding
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
		base64:    base64.URLEncoding,
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

	return st.base64.EncodeToString(buff.Bytes()), nil
}

// TokenDecode decodes given base64 encoded and encrypted token into
// SessionState.
func (st *SessionTokenizer) TokenDecode(token string) (*SessionState, error) {
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

// jsonResponse sends a JSON response with given status code.
func jsonResponse(w http.ResponseWriter, code int, i interface{}) error {
	b, err := json.Marshal(i)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	w.Write(b)
	return nil
}

type responseWrapper struct {
	Data  interface{} `json:"data,omitempty"`
	Error interface{} `json:"error,omitempty"`
	Debug interface{} `json:"debug,omitempty"`
}

type errorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// SessionRequired is http middleware which checks for presence of session
// state in current request. It return  request without auth header set
// or with invalid value of session token.
//
// If token is present, SessionRequired saves given token within request
// context. It can be retrieved with SessionContextState function.
func SessionRequired(cs *SessionCookieStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			state, err := cs.SessionState(r)
			if err != nil {
				jsonResponse(w, http.StatusUnauthorized, responseWrapper{
					Error: errorResponse{
						Code:    http.StatusUnauthorized,
						Message: "You are not authorized to access these resources.",
					},
				})
				return
			}

			ctx := context.WithValue(r.Context(), sessionStateKey, state)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type sessionKey string

const sessionStateKey sessionKey = "__session_state"

// SessionContextState retrieves session state from context. It
// returns nil context, if there is no session state saved within
// context.
func SessionContextState(ctx context.Context) *SessionState {
	res, ok := ctx.Value(sessionStateKey).(*SessionState)
	if !ok {
		return nil
	}
	return res
}

const (
	sessionCookieKey      = "SzmaterlokSession"
	sessionExpirationDate = time.Hour * 24 * 7
)

// SessionCookieSetRequest contains dependencies and arguments
// for saving session cookie.
type SessionCookieSetRequest struct {
	Writer    http.ResponseWriter
	Request   *http.Request
	Tokenizer *SessionTokenizer
	State     SessionState
	Clock
}

var ErrSessionStateExpire = errors.New("session state expired")

// SessionCookieStore handles save and read operation of session
// state token within http cookies.
type SessionCookieStore struct {
	// ExpirationTime of http cookie. It can differ from session
	// state expiration date, but session state's one is more
	// important. Valid cookie with expired session state will be
	// invalid.
	ExpirationTime time.Duration

	// Tokenizer handles encoding and decoding of session state.
	Tokenizer *SessionTokenizer

	// Clock returns current time.
	Clock
}

// SessionState returns current session state retrieved from http cookies.
func (cs *SessionCookieStore) SessionState(r *http.Request) (*SessionState, error) {
	c, err := r.Cookie(sessionCookieKey)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to retrieve with %s cookie from req: %w",
			sessionCookieKey, err,
		)
	}

	state, err := cs.Tokenizer.TokenDecode(c.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to decode cookie: %w", err)
	}

	if state.ExpireAt.Before(cs.Now()) {
		return nil, ErrSessionStateExpire
	}

	return state, nil
}

// SaveSessionState overwrites szmaterlok session cookie with given
// SessionState.
func (cs *SessionCookieStore) SaveSessionState(
	w http.ResponseWriter, s SessionState,
) error {
	token, err := cs.Tokenizer.TokenEncode(s)
	if err != nil {
		return fmt.Errorf("failed to tokenize state: %w", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieKey,
		Value:    token,
		Path:     "/",
		Expires:  cs.Now().Add(cs.ExpirationTime),
		HttpOnly: true,
	})
	return nil
}

// ClearState deletes current session state stored in http cookies.
func (cs *SessionCookieStore) ClearState(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieKey,
		Value:    "",
		Path:     "/",
		Expires:  cs.Now().Add(-1 * time.Second),
		HttpOnly: true,
	})
}

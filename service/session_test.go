package service

import (
	"testing"
	"time"

	"github.com/matryer/is"
)

func TestSessionAgeTokenizer(t *testing.T) {
	is := is.New(t)

	pass := "secret_password"
	tokenizerA, err := NewSessionAgeTokenizer(pass)
	expirationTime := time.Hour * 24 * 7

	is.NoErr(err)

	now, err := time.Parse(time.ANSIC, "Thu Mar 17 21:23:59 2022")
	is.NoErr(err)

	wantState := SessionState{
		Nickname:  "karol",
		ID:        "uniqueid",
		CreatedAt: now,
		ExpireAt:  now.Add(expirationTime),
	}

	token, err := tokenizerA.TokenEncode(wantState)
	is.NoErr(err)
	is.True(token != "")

	tokenizerB, err := NewSessionAgeTokenizer(pass)
	gotState, err := tokenizerB.TokenDecode(token)
	is.NoErr(err)
	is.True(gotState != nil)

	is.Equal(*gotState, wantState)
}

func TestAESTokenizer(t *testing.T) {
	is := is.New(t)

	pass := []byte("veibiequohy2eshaerohHoghootae1ku")
	expirationTime := time.Hour * 24 * 7
	tokenizer, err := NewSessionAESTokenizer(pass)
	is.NoErr(err)

	now, err := time.Parse(time.ANSIC, "Thu Mar 17 21:23:59 2022")
	is.NoErr(err)

	wantState := SessionState{
		Nickname:  "karol",
		ID:        "uniqueid",
		CreatedAt: now,
		ExpireAt:  now.Add(expirationTime),
	}

	token, err := tokenizer.TokenEncode(wantState)
	is.NoErr(err)
	is.True(token != "")

	gotState, err := tokenizer.TokenDecode(token)
	is.NoErr(err)
	is.True(gotState != nil)

	is.Equal(*gotState, wantState)
}

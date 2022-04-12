package service

import (
	"context"
	"sort"
	"testing"

	"github.com/matryer/is"
)

func TestStateOnlineUsers(t *testing.T) {
	t.Run("PushChatUser", func(t *testing.T) {
		ctx := context.TODO()
		is := is.New(t)

		state := NewStateOnlineUsers()
		is.True(state != nil)

		err := state.PushChatUser(ctx, StateChatUser{
			ID:       "1",
			Nickname: "nickname",
		})
		is.NoErr(err)

		u, ok := state.state["1"]
		is.True(ok)
		is.Equal(u.ID, "1")
		is.Equal(u.Nickname, "nickname")
	})

	t.Run("AllChatUsers", func(t *testing.T) {
		ctx := context.TODO()
		is := is.New(t)

		state := NewStateOnlineUsers()
		is.True(state != nil)

		want := []OnlineChatUser{
			{
				ID:       "1",
				Nickname: "Nickname1",
			},
			{
				ID:       "2",
				Nickname: "Nickname2",
			},
			{
				ID:       "3",
				Nickname: "Nickname3",
			},
		}

		for _, u := range want {
			state.state[u.ID] = StateChatUser{
				ID:       u.ID,
				Nickname: u.Nickname,
			}
		}

		got, err := state.AllChatUsers(ctx)
		is.NoErr(err)
		is.True(len(got) != 0)

		sort.Slice(got, func(i, j int) bool {
			return got[i].ID < got[j].ID
		})
		is.Equal(got, want)
	})

	t.Run("RemoveChatUser", func(t *testing.T) {
		ctx := context.TODO()
		is := is.New(t)

		state := NewStateOnlineUsers()
		is.True(state != nil)

		state.state["1"] = StateChatUser{
			ID:       "1",
			Nickname: "nickname",
		}

		err := state.RemoveChatUser(ctx, "1")
		is.NoErr(err)

		_, ok := state.state["1"]
		is.True(!ok)
	})
}

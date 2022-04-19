package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

// StateChatUser contains data of single user who is
// currently logged in into the chat.
type StateChatUser struct {
	ID       string
	Nickname string
}

// StateOnlineUsers contains data for users, which
// are currently using chat.
type StateOnlineUsers struct {
	mtx   *sync.Mutex
	state map[string]StateChatUser
}

// NewStateOnlineUsers is constructor for StateOnlineUsers. Using
// NewStateOnlineUsers is the only safe way to construct StateOnlineUsers.
func NewStateOnlineUsers() *StateOnlineUsers {
	return &StateOnlineUsers{
		mtx:   &sync.Mutex{},
		state: map[string]StateChatUser{},
	}
}

// AllChatUsers returns all users which are using currently chat.
func (s *StateOnlineUsers) AllChatUsers(ctx context.Context) ([]OnlineChatUser, error) {
	res := []OnlineChatUser{}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	for _, u := range s.state {
		res = append(res, OnlineChatUser{
			ID:       u.ID,
			Nickname: u.Nickname,
		})
	}

	return res, nil
}

// PushChatUser saves data of user which is logging in.
func (s *StateOnlineUsers) PushChatUser(ctx context.Context, u StateChatUser) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.state[u.ID] = u

	return nil
}

var ErrNoSuchUser = errors.New("state: there is no such user")

// RemoveChatUser removes user with given id from state storage.
func (s *StateOnlineUsers) RemoveChatUser(ctx context.Context, id string) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	_, ok := s.state[id]
	if !ok {
		return ErrNoSuchUser
	}

	delete(s.state, id)

	return nil
}

// StateUserJoinHook adds new user to state online users storage when such
// joins the chat.
func StateUserJoinHook(log *logrus.Logger, s *StateOnlineUsers) BridgeEventHandlerFunc {
	return func(ctx context.Context, evt BridgeEvent) {
		evtData := &EventUserJoin{}

		if err := json.Unmarshal(evt.Data, evtData); err != nil {
			log.WithFields(logrus.Fields{
				"scope":   "StateUserJoinHook",
				"reqID":   evt.Headers.Get(bridgeRequestIDHeaderVar),
				"eventID": evt.ID,
				"error":   err.Error(),
			}).Errorln("Failed to unmarshal EventUserJoin data.")
			return
		}

		if err := s.PushChatUser(ctx, StateChatUser{
			ID:       evtData.User.ID,
			Nickname: evtData.User.Nickname,
		}); err != nil {
			log.WithFields(logrus.Fields{
				"scope":   "StateUserJoinHook",
				"reqID":   evt.Headers.Get(bridgeRequestIDHeaderVar),
				"eventID": evt.ID,
				"error":   err.Error(),
			}).Errorln("Failed to push chat user.")
		}
	}
}

// StateUserLeftHook removes user from state online users storage when such
// lefts the chat.
func StateUserLeftHook(log *logrus.Logger, s *StateOnlineUsers) BridgeEventHandlerFunc {
	return func(ctx context.Context, evt BridgeEvent) {
		evtData := &EventUserLeft{}

		if err := json.Unmarshal(evt.Data, evtData); err != nil {
			log.WithFields(logrus.Fields{
				"scope":   "StateUserLeftHook",
				"reqID":   evt.Headers.Get(bridgeRequestIDHeaderVar),
				"eventID": evt.ID,
				"error":   err.Error(),
			}).Errorln("Failed to unmarshal EventUserLeft data.")
			return
		}

		if err := s.RemoveChatUser(ctx, evtData.User.ID); err != nil {
			log.WithFields(logrus.Fields{
				"scope":   "StateUserLeftHook",
				"reqID":   evt.Headers.Get(bridgeRequestIDHeaderVar),
				"eventID": evt.ID,
				"error":   err.Error(),
			}).Errorln("Failed to remove user from chat.")
		}

	}
}

// StateArchive stores events from past. With state archive application
// is able to rebuild its state.
type StateArchive interface {
	// Events sends all events from state archive through given channels
	// grouped by their creation date.
	Events(context.Context, chan<- BridgeEvent) error
}

// StateBuilder rebuilds state of application with events from
// state archive.
type StateBuilder struct {
	// Archive stores past events.
	Archive StateArchive

	// Handler rebuilds state by applying hook to events
	// from archive.
	Handler BridgeEventHandler
}

// Rebuild whole state of application.
func (sb *StateBuilder) Rebuild(ctx context.Context) error {
	errc := make(chan error, 1)
	evtc := make(chan BridgeEvent)

	go func() {
		defer close(evtc)
		errc <- sb.Archive.Events(ctx, evtc)
	}()

	for evt := range evtc {
		sb.Handler.EventHook(ctx, evt)
	}

	if err := <-errc; err != nil {
		return fmt.Errorf("failed to read from archive: %w", err)
	}

	return nil
}

package main

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewMessageStore(t *testing.T) {
	api := &plugintest.API{}
	store := NewMessageStore(api)

	assert.NotNil(t, store)
	assert.Equal(t, api, store.api)
}

func TestMessageStoreKVKey(t *testing.T) {
	api := &plugintest.API{}
	store := NewMessageStore(api)

	tests := []struct {
		name      string
		sessionID string
		wantKey   string
	}{
		{
			name:      "normal session ID",
			sessionID: "session123",
			wantKey:   "messages_session123",
		},
		{
			name:      "empty session ID",
			sessionID: "",
			wantKey:   "messages_",
		},
		{
			name:      "session ID with special chars",
			sessionID: "user_123_channel_456",
			wantKey:   "messages_user_123_channel_456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := store.kvKey(tt.sessionID)
			assert.Equal(t, tt.wantKey, key)
		})
	}
}

func TestMessageStoreGetMessages(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		kvData    []byte
		kvErr     *model.AppError
		want      []StoredMessage
		wantErr   bool
	}{
		{
			name:      "empty messages",
			sessionID: "session1",
			kvData:    nil,
			kvErr:     nil,
			want:      []StoredMessage{},
			wantErr:   false,
		},
		{
			name:      "valid messages",
			sessionID: "session2",
			kvData: func() []byte {
				sm := SessionMessages{
					SessionID: "session2",
					Messages: []StoredMessage{
						{
							ID:        "session2_0",
							SessionID: "session2",
							Role:      "user",
							Content:   "Hello",
							Timestamp: time.Now(),
						},
						{
							ID:        "session2_1",
							SessionID: "session2",
							Role:      "assistant",
							Content:   "Hi there!",
							Timestamp: time.Now(),
						},
					},
					UpdatedAt: time.Now(),
				}
				data, _ := json.Marshal(sm)
				return data
			}(),
			kvErr:   nil,
			want:    []StoredMessage{{}, {}},
			wantErr: false,
		},
		{
			name:      "KV get error",
			sessionID: "session3",
			kvData:    nil,
			kvErr:     model.NewAppError("test", "test.error", nil, "", 500),
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "invalid JSON",
			sessionID: "session4",
			kvData:    []byte("invalid json"),
			kvErr:     nil,
			want:      nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &plugintest.API{}
			store := NewMessageStore(api)

			api.On("KVGet", fmt.Sprintf("messages_%s", tt.sessionID)).Return(tt.kvData, tt.kvErr)

			got, err := store.GetMessages(tt.sessionID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.want == nil {
					assert.Nil(t, got)
				} else {
					assert.Len(t, got, len(tt.want))
				}
			}

			api.AssertExpectations(t)
		})
	}
}

func TestMessageStoreAddMessage(t *testing.T) {
	tests := []struct {
		name           string
		sessionID      string
		role           string
		content        string
		existingData   []byte
		getErr         *model.AppError
		setErr         *model.AppError
		wantErr        bool
		wantMessageID  string
		wantRole       string
		wantContent    string
	}{
		{
			name:          "add first message",
			sessionID:     "session1",
			role:          "user",
			content:       "Hello",
			existingData:  nil,
			getErr:        nil,
			setErr:        nil,
			wantErr:       false,
			wantMessageID: "session1_0",
			wantRole:      "user",
			wantContent:   "Hello",
		},
		{
			name:      "add second message",
			sessionID: "session2",
			role:      "assistant",
			content:   "Hi there!",
			existingData: func() []byte {
				sm := SessionMessages{
					SessionID: "session2",
					Messages: []StoredMessage{
						{
							ID:        "session2_0",
							SessionID: "session2",
							Role:      "user",
							Content:   "Hello",
							Timestamp: time.Now(),
						},
					},
					UpdatedAt: time.Now(),
				}
				data, _ := json.Marshal(sm)
				return data
			}(),
			getErr:        nil,
			setErr:        nil,
			wantErr:       false,
			wantMessageID: "session2_1",
			wantRole:      "assistant",
			wantContent:   "Hi there!",
		},
		{
			name:         "KV set error",
			sessionID:    "session3",
			role:         "user",
			content:      "Test",
			existingData: nil,
			getErr:       nil,
			setErr:       model.NewAppError("test", "test.error", nil, "", 500),
			wantErr:      true,
		},
		{
			name:          "get error but still works",
			sessionID:     "session4",
			role:          "user",
			content:       "Test",
			existingData:  nil,
			getErr:        model.NewAppError("test", "test.error", nil, "", 500),
			setErr:        nil,
			wantErr:       false,
			wantMessageID: "session4_0",
			wantRole:      "user",
			wantContent:   "Test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &plugintest.API{}
			store := NewMessageStore(api)

			kvKey := fmt.Sprintf("messages_%s", tt.sessionID)
			api.On("KVGet", kvKey).Return(tt.existingData, tt.getErr)
			if tt.setErr != nil {
				api.On("KVSet", kvKey, mock.Anything).Return(tt.setErr)
			} else {
				api.On("KVSet", kvKey, mock.Anything).Return(nil)
			}

			msg, err := store.AddMessage(tt.sessionID, tt.role, tt.content)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, msg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, msg)
				if msg != nil {
					assert.Equal(t, tt.wantMessageID, msg.ID)
					assert.Equal(t, tt.sessionID, msg.SessionID)
					assert.Equal(t, tt.wantRole, msg.Role)
					assert.Equal(t, tt.wantContent, msg.Content)
					assert.False(t, msg.Timestamp.IsZero())
				}
			}

			api.AssertExpectations(t)
		})
	}
}

func TestMessageStoreDeleteSessionMessages(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		deleteErr *model.AppError
		wantErr   bool
	}{
		{
			name:      "successful delete",
			sessionID: "session1",
			deleteErr: nil,
			wantErr:   false,
		},
		{
			name:      "delete error",
			sessionID: "session2",
			deleteErr: model.NewAppError("test", "test.error", nil, "", 500),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &plugintest.API{}
			store := NewMessageStore(api)

			kvKey := fmt.Sprintf("messages_%s", tt.sessionID)
			api.On("KVDelete", kvKey).Return(tt.deleteErr)

			err := store.DeleteSessionMessages(tt.sessionID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			api.AssertExpectations(t)
		})
	}
}

func TestMessageStoreGetMessageCount(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		kvData    []byte
		kvErr     *model.AppError
		wantCount int
		wantErr   bool
	}{
		{
			name:      "empty messages",
			sessionID: "session1",
			kvData:    nil,
			kvErr:     nil,
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "multiple messages",
			sessionID: "session2",
			kvData: func() []byte {
				sm := SessionMessages{
					SessionID: "session2",
					Messages: []StoredMessage{
						{ID: "1", Role: "user", Content: "Hello"},
						{ID: "2", Role: "assistant", Content: "Hi"},
						{ID: "3", Role: "user", Content: "How are you?"},
					},
					UpdatedAt: time.Now(),
				}
				data, _ := json.Marshal(sm)
				return data
			}(),
			kvErr:     nil,
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:      "get error",
			sessionID: "session3",
			kvData:    nil,
			kvErr:     model.NewAppError("test", "test.error", nil, "", 500),
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &plugintest.API{}
			store := NewMessageStore(api)

			api.On("KVGet", fmt.Sprintf("messages_%s", tt.sessionID)).Return(tt.kvData, tt.kvErr)

			count, err := store.GetMessageCount(tt.sessionID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCount, count)
			}

			api.AssertExpectations(t)
		})
	}
}

func TestMessageStoreGetLastMessage(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		kvData    []byte
		kvErr     *model.AppError
		wantMsg   *StoredMessage
		wantErr   bool
	}{
		{
			name:      "empty messages",
			sessionID: "session1",
			kvData:    nil,
			kvErr:     nil,
			wantMsg:   nil,
			wantErr:   false,
		},
		{
			name:      "single message",
			sessionID: "session2",
			kvData: func() []byte {
				sm := SessionMessages{
					SessionID: "session2",
					Messages: []StoredMessage{
						{ID: "1", Role: "user", Content: "Hello"},
					},
					UpdatedAt: time.Now(),
				}
				data, _ := json.Marshal(sm)
				return data
			}(),
			kvErr: nil,
			wantMsg: &StoredMessage{
				ID:      "1",
				Role:    "user",
				Content: "Hello",
			},
			wantErr: false,
		},
		{
			name:      "multiple messages",
			sessionID: "session3",
			kvData: func() []byte {
				sm := SessionMessages{
					SessionID: "session3",
					Messages: []StoredMessage{
						{ID: "1", Role: "user", Content: "Hello"},
						{ID: "2", Role: "assistant", Content: "Hi"},
						{ID: "3", Role: "user", Content: "Last"},
					},
					UpdatedAt: time.Now(),
				}
				data, _ := json.Marshal(sm)
				return data
			}(),
			kvErr: nil,
			wantMsg: &StoredMessage{
				ID:      "3",
				Role:    "user",
				Content: "Last",
			},
			wantErr: false,
		},
		{
			name:      "get error",
			sessionID: "session4",
			kvData:    nil,
			kvErr:     model.NewAppError("test", "test.error", nil, "", 500),
			wantMsg:   nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &plugintest.API{}
			store := NewMessageStore(api)

			api.On("KVGet", fmt.Sprintf("messages_%s", tt.sessionID)).Return(tt.kvData, tt.kvErr)

			msg, err := store.GetLastMessage(tt.sessionID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.wantMsg == nil {
					assert.Nil(t, msg)
				} else {
					assert.NotNil(t, msg)
					assert.Equal(t, tt.wantMsg.ID, msg.ID)
					assert.Equal(t, tt.wantMsg.Role, msg.Role)
					assert.Equal(t, tt.wantMsg.Content, msg.Content)
				}
			}

			api.AssertExpectations(t)
		})
	}
}

func TestMessageStoreGetMessagesByRole(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		role      string
		kvData    []byte
		kvErr     *model.AppError
		wantCount int
		wantErr   bool
	}{
		{
			name:      "empty messages",
			sessionID: "session1",
			role:      "user",
			kvData:    nil,
			kvErr:     nil,
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "filter user messages",
			sessionID: "session2",
			role:      "user",
			kvData: func() []byte {
				sm := SessionMessages{
					SessionID: "session2",
					Messages: []StoredMessage{
						{ID: "1", Role: "user", Content: "Hello"},
						{ID: "2", Role: "assistant", Content: "Hi"},
						{ID: "3", Role: "user", Content: "How are you?"},
						{ID: "4", Role: "assistant", Content: "Good!"},
						{ID: "5", Role: "user", Content: "Great"},
					},
					UpdatedAt: time.Now(),
				}
				data, _ := json.Marshal(sm)
				return data
			}(),
			kvErr:     nil,
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:      "filter assistant messages",
			sessionID: "session3",
			role:      "assistant",
			kvData: func() []byte {
				sm := SessionMessages{
					SessionID: "session3",
					Messages: []StoredMessage{
						{ID: "1", Role: "user", Content: "Hello"},
						{ID: "2", Role: "assistant", Content: "Hi"},
						{ID: "3", Role: "user", Content: "How are you?"},
						{ID: "4", Role: "assistant", Content: "Good!"},
					},
					UpdatedAt: time.Now(),
				}
				data, _ := json.Marshal(sm)
				return data
			}(),
			kvErr:     nil,
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "no matching role",
			sessionID: "session4",
			role:      "system",
			kvData: func() []byte {
				sm := SessionMessages{
					SessionID: "session4",
					Messages: []StoredMessage{
						{ID: "1", Role: "user", Content: "Hello"},
						{ID: "2", Role: "assistant", Content: "Hi"},
					},
					UpdatedAt: time.Now(),
				}
				data, _ := json.Marshal(sm)
				return data
			}(),
			kvErr:     nil,
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "get error",
			sessionID: "session5",
			role:      "user",
			kvData:    nil,
			kvErr:     model.NewAppError("test", "test.error", nil, "", 500),
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &plugintest.API{}
			store := NewMessageStore(api)

			api.On("KVGet", fmt.Sprintf("messages_%s", tt.sessionID)).Return(tt.kvData, tt.kvErr)

			messages, err := store.GetMessagesByRole(tt.sessionID, tt.role)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, messages, tt.wantCount)
				// Verify all returned messages have the correct role
				for _, msg := range messages {
					assert.Equal(t, tt.role, msg.Role)
				}
			}

			api.AssertExpectations(t)
		})
	}
}

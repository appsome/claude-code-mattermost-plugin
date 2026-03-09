package main

import (
	"encoding/json"
	"testing"

	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetActiveSession_NoSession(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)
	
	// Return nil for KVGet (no session)
	api.On("KVGet", "session_channel1").Return(nil, nil)
	
	defer api.AssertExpectations(t)

	session, err := p.GetActiveSession("channel1")
	assert.NoError(t, err)
	assert.Nil(t, session)
}

func TestGetActiveSession_ExistingSession(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)
	
	// Create a session object
	expectedSession := &ChannelSession{
		SessionID:     "session123",
		ProjectPath:   "/tmp/test",
		UserID:        "user1",
		CreatedAt:     1000000,
		LastMessageAt: 1000000,
	}
	
	// Marshal it to JSON
	data, _ := json.Marshal(expectedSession)
	
	// Mock KVGet to return the session
	api.On("KVGet", "session_channel1").Return(data, nil)
	
	defer api.AssertExpectations(t)

	session, err := p.GetActiveSession("channel1")
	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, "session123", session.SessionID)
	assert.Equal(t, "/tmp/test", session.ProjectPath)
	assert.Equal(t, "user1", session.UserID)
}

func TestSaveSession(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)
	
	session := &ChannelSession{
		SessionID:     "session123",
		ProjectPath:   "/tmp/test",
		UserID:        "user1",
		CreatedAt:     1000000,
		LastMessageAt: 1000000,
	}
	
	// Mock KVSet
	api.On("KVSet", "session_channel1", mock.Anything).Return(nil)
	
	defer api.AssertExpectations(t)

	err := p.SaveSession("channel1", session)
	assert.NoError(t, err)
}

func TestDeleteSession(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)
	
	// Mock KVDelete
	api.On("KVDelete", "session_channel1").Return(nil)
	
	defer api.AssertExpectations(t)

	err := p.DeleteSession("channel1")
	assert.NoError(t, err)
}

func TestUpdateSessionLastMessage(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)
	
	session := &ChannelSession{
		SessionID:     "session123",
		ProjectPath:   "/tmp/test",
		UserID:        "user1",
		CreatedAt:     1000000,
		LastMessageAt: 1000000,
	}
	
	data, _ := json.Marshal(session)
	
	// Mock KVGet to return existing session
	api.On("KVGet", "session_channel1").Return(data, nil)
	
	// Mock KVSet to save updated session
	api.On("KVSet", "session_channel1", mock.Anything).Return(nil)
	
	defer api.AssertExpectations(t)

	err := p.UpdateSessionLastMessage("channel1")
	assert.NoError(t, err)
}

func TestGetSessionForChannel_NoSession(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)
	
	// No session
	api.On("KVGet", "session_channel1").Return(nil, nil)
	
	defer api.AssertExpectations(t)

	sessionID := p.GetSessionForChannel("channel1")
	assert.Empty(t, sessionID)
}

func TestGetSessionForChannel_ExistingSession(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)
	
	session := &ChannelSession{
		SessionID:   "session123",
		ProjectPath: "/tmp/test",
		UserID:      "user1",
	}
	
	data, _ := json.Marshal(session)
	api.On("KVGet", "session_channel1").Return(data, nil)
	
	defer api.AssertExpectations(t)

	sessionID := p.GetSessionForChannel("channel1")
	assert.Equal(t, "session123", sessionID)
}

func TestSessionManager_CreateSession_SkippedForNow(t *testing.T) {
	// Skipped: CreateSession and StopSession tests require interface refactoring
	// Plugin.bridgeClient needs to be an interface to allow mocking
	t.Skip("Requires refactoring Plugin.bridgeClient to use an interface")
}

// CreateSession and StopSession tests skipped - require interface refactoring
// These functions call bridgeClient methods which cannot be easily mocked without
// refactoring Plugin.bridgeClient to use an interface instead of a concrete type

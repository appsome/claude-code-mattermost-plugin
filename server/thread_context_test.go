package main

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
)

func TestGetThreadContext_EmptyThread(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)
	
	// Mock GetChannel
	channel := &model.Channel{
		Id:   "channel1",
		Name: "test-channel",
	}
	api.On("GetChannel", "channel1").Return(channel, nil)
	
	// Mock GetPostThread to return truly empty thread (no posts)
	postList := &model.PostList{
		Order: []string{},
		Posts: map[string]*model.Post{},
	}
	api.On("GetPostThread", "root123").Return(postList, nil)
	
	defer api.AssertExpectations(t)

	_, err := p.GetThreadContext("root123", "channel1", 50)
	// Should error because thread is empty
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "thread is empty")
}

func TestGetThreadContext_SinglePost(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)
	
	// Mock GetChannel
	channel := &model.Channel{
		Id:   "channel1",
		Name: "test-channel",
	}
	api.On("GetChannel", "channel1").Return(channel, nil)
	
	// Create a single post
	post := &model.Post{
		Id:        "post1",
		UserId:    "user1",
		ChannelId: "channel1",
		Message:   "Hello world",
		CreateAt:  1000000000,
	}
	
	postList := &model.PostList{
		Order: []string{"post1"},
		Posts: map[string]*model.Post{
			"post1": post,
		},
	}
	
	// Mock user lookup
	user := &model.User{
		Id:       "user1",
		Username: "testuser",
	}
	
	api.On("GetPostThread", "post1").Return(postList, nil)
	api.On("GetUser", "user1").Return(user, nil)
	
	defer api.AssertExpectations(t)

	context, err := p.GetThreadContext("post1", "channel1", 50)
	assert.NoError(t, err)
	assert.NotNil(t, context)
	assert.Contains(t, context.Content, "testuser")
	assert.Contains(t, context.Content, "Hello world")
	assert.Equal(t, 1, context.MessageCount)
	assert.Contains(t, context.Participants, "@testuser")
}

func TestGetThreadContext_MultipleMessages(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)
	
	// Mock GetChannel
	channel := &model.Channel{
		Id:   "channel1",
		Name: "test-channel",
	}
	api.On("GetChannel", "channel1").Return(channel, nil)
	
	// Create multiple posts
	post1 := &model.Post{
		Id:        "root123",
		UserId:    "user1",
		ChannelId: "channel1",
		Message:   "First message",
		CreateAt:  1000000000,
	}
	
	post2 := &model.Post{
		Id:        "post2",
		UserId:    "user2",
		ChannelId: "channel1",
		Message:   "Second message",
		CreateAt:  2000000000,
	}
	
	postList := &model.PostList{
		Order: []string{"root123", "post2"},
		Posts: map[string]*model.Post{
			"root123": post1,
			"post2":   post2,
		},
	}
	
	user1 := &model.User{Id: "user1", Username: "alice"}
	user2 := &model.User{Id: "user2", Username: "bob"}
	
	api.On("GetPostThread", "root123").Return(postList, nil)
	api.On("GetUser", "user1").Return(user1, nil)
	api.On("GetUser", "user2").Return(user2, nil)
	
	defer api.AssertExpectations(t)

	context, err := p.GetThreadContext("root123", "channel1", 50)
	assert.NoError(t, err)
	assert.NotNil(t, context)
	assert.Contains(t, context.Content, "alice")
	assert.Contains(t, context.Content, "First message")
	assert.Contains(t, context.Content, "bob")
	assert.Contains(t, context.Content, "Second message")
	assert.Equal(t, 2, context.MessageCount)
	assert.Len(t, context.Participants, 2)
}

func TestGetThreadContext_MaxMessagesLimit(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)
	
	// Mock GetChannel
	channel := &model.Channel{
		Id:   "channel1",
		Name: "test-channel",
	}
	api.On("GetChannel", "channel1").Return(channel, nil)
	
	// Create root post + 9 more posts (10 total)
	posts := make(map[string]*model.Post)
	order := make([]string, 10)
	
	// Root post
	posts["root123"] = &model.Post{
		Id:        "root123",
		UserId:    "user1",
		ChannelId: "channel1",
		Message:   "Root message",
		CreateAt:  1000000000,
	}
	order[0] = "root123"
	
	// Add 9 more posts
	for i := 1; i < 10; i++ {
		postID := model.NewId()
		posts[postID] = &model.Post{
			Id:        postID,
			UserId:    "user1",
			ChannelId: "channel1",
			Message:   "Message " + string(rune('0'+i)),
			CreateAt:  int64(1000000000 + i*1000),
		}
		order[i] = postID
	}
	
	postList := &model.PostList{
		Order: order,
		Posts: posts,
	}
	
	user := &model.User{Id: "user1", Username: "testuser"}
	
	api.On("GetPostThread", "root123").Return(postList, nil)
	// GetUser will be called once for each of the last 5 messages (all same user)
	// But since they're all the same user, it will still be called 5 times
	api.On("GetUser", "user1").Return(user, nil).Times(5)
	
	defer api.AssertExpectations(t)

	// Limit to 5 messages
	context, err := p.GetThreadContext("root123", "channel1", 5)
	assert.NoError(t, err)
	assert.NotNil(t, context)
	assert.Equal(t, 5, context.MessageCount)
}

func TestGetThreadContext_WithFileAttachments(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)
	
	// Mock GetChannel
	channel := &model.Channel{
		Id:   "channel1",
		Name: "test-channel",
	}
	api.On("GetChannel", "channel1").Return(channel, nil)
	
	// Create post with file attachments
	post := &model.Post{
		Id:        "post1",
		UserId:    "user1",
		ChannelId: "channel1",
		Message:   "Check out these files",
		CreateAt:  1000000000,
		FileIds:   []string{"file1", "file2"},
	}
	
	postList := &model.PostList{
		Order: []string{"post1"},
		Posts: map[string]*model.Post{
			"post1": post,
		},
	}
	
	user := &model.User{Id: "user1", Username: "testuser"}
	
	api.On("GetPostThread", "post1").Return(postList, nil)
	api.On("GetUser", "user1").Return(user, nil)
	
	defer api.AssertExpectations(t)

	context, err := p.GetThreadContext("post1", "channel1", 50)
	assert.NoError(t, err)
	assert.NotNil(t, context)
	assert.Contains(t, context.Content, "file(s) attached")
}

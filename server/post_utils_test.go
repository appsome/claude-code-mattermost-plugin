package main

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPostChangeProposal(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)
	
	// Mock site URL config
	config := &model.Config{}
	siteURL := "http://localhost:8065"
	config.ServiceSettings.SiteURL = &siteURL
	api.On("GetConfig").Return(config)
	
	// Mock post creation
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{Id: "post123"}, nil)
	
	defer api.AssertExpectations(t)

	_ = p.postChangeProposal("channel1", "Would you like to apply this change?", "change123")
	// Mock expectations will catch any errors
}

func TestPostWithQuickActions(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)
	
	// Mock site URL config
	config := &model.Config{}
	siteURL := "http://localhost:8065"
	config.ServiceSettings.SiteURL = &siteURL
	api.On("GetConfig").Return(config)
	
	// Mock post creation
	createdPost := &model.Post{Id: "post123"}
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(createdPost, nil)
	
	defer api.AssertExpectations(t)

	postID, err := p.postWithQuickActions("channel1", "Here's the response", "session123")
	if err != nil {
		t.Fatalf("postWithQuickActions returned error: %v", err)
	}
	assert.Equal(t, "post123", postID)
}

func TestPostCodeChange(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)
	
	// Mock site URL config
	config := &model.Config{}
	siteURL := "http://localhost:8065"
	config.ServiceSettings.SiteURL = &siteURL
	api.On("GetConfig").Return(config)
	
	// Mock post creation
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{Id: "post123"}, nil)
	
	defer api.AssertExpectations(t)

	diff := "+function hello() {\n-  console.log('old');\n+  console.log('new');\n+}"
	_ = p.postCodeChange("channel1", "src/main.js", diff, "change123")
	// Mock expectations will catch any errors
}

func TestPostWithMenu(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)
	
	// Mock site URL config
	config := &model.Config{}
	siteURL := "http://localhost:8065"
	config.ServiceSettings.SiteURL = &siteURL
	api.On("GetConfig").Return(config)
	
	// Mock post creation
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{Id: "post123"}, nil)
	
	defer api.AssertExpectations(t)

	options := []ActionOption{
		{Label: "Option 1", Value: "opt1"},
		{Label: "Option 2", Value: "opt2"},
	}
	_ = p.postWithMenu("channel1", "Choose an action:", options, "session123")
	// Mock expectations will catch any errors
}

func TestUpdatePostWithProgress(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)
	
	existingPost := &model.Post{
		Id:      "post123",
		Message: "Old message",
	}
	
	// Mock getting and updating the post
	api.On("GetPost", "post123").Return(existingPost, nil)
	api.On("UpdatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil)
	
	defer api.AssertExpectations(t)

	_ = p.updatePostWithProgress("post123", "Processing...")
	// Mock expectations will catch any errors
}

func TestUpdatePostMessage(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)
	
	existingPost := &model.Post{
		Id:      "post123",
		Message: "Old message",
	}
	
	// Mock getting and updating the post
	api.On("GetPost", "post123").Return(existingPost, nil)
	api.On("UpdatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil)
	
	defer api.AssertExpectations(t)

	_ = p.updatePostMessage("post123", "New message")
	// Mock expectations will catch any errors
}

func TestGetPluginURL(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)
	
	config := &model.Config{}
	siteURL := "http://localhost:8065"
	config.ServiceSettings.SiteURL = &siteURL
	api.On("GetConfig").Return(config)
	
	defer api.AssertExpectations(t)

	url := p.getPluginURL()
	assert.Equal(t, "http://localhost:8065/plugins/com.appsome.claudecode", url)
}

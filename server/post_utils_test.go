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

	err := p.postChangeProposal("channel1", "Would you like to apply this change?", "change123")
	assert.NoError(t, err)
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
	assert.NoError(t, err)
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
	err := p.postCodeChange("channel1", "src/main.js", diff, "change123")
	assert.NoError(t, err)
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
	err := p.postWithMenu("channel1", "Choose an action:", options, "session123")
	assert.NoError(t, err)
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

	err := p.updatePostWithProgress("post123", "Processing...")
	assert.NoError(t, err)
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

	err := p.updatePostMessage("post123", "New message")
	assert.NoError(t, err)
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

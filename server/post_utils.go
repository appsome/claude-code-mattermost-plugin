package main

import (
	"fmt"

	"github.com/mattermost/mattermost/server/public/model"
)

// postChangeProposal creates a post with approve/reject/modify buttons for code changes
func (p *Plugin) postChangeProposal(channelID, content string, changeID string) error {
	attachment := &model.SlackAttachment{
		Text: content,
		Actions: []*model.PostAction{
			{
				Id:    "approve_" + changeID,
				Name:  "✅ Approve",
				Type:  "button",
				Style: "primary",
				Integration: &model.PostActionIntegration{
					URL: p.getPluginURL() + "/api/action/approve",
					Context: map[string]interface{}{
						"change_id": changeID,
					},
				},
			},
			{
				Id:    "reject_" + changeID,
				Name:  "❌ Reject",
				Type:  "button",
				Style: "danger",
				Integration: &model.PostActionIntegration{
					URL: p.getPluginURL() + "/api/action/reject",
					Context: map[string]interface{}{
						"change_id": changeID,
					},
				},
			},
			{
				Id:   "modify_" + changeID,
				Name: "✏️ Modify",
				Type: "button",
				Integration: &model.PostActionIntegration{
					URL: p.getPluginURL() + "/api/action/modify",
					Context: map[string]interface{}{
						"change_id": changeID,
					},
				},
			},
		},
	}

	post := &model.Post{
		ChannelId: channelID,
		UserId:    p.botUserID,
		Message:   "Claude Code has proposed changes:",
		Props: model.StringInterface{
			"attachments": []*model.SlackAttachment{attachment},
		},
	}

	_, err := p.API.CreatePost(post)
	return err
}

// postWithQuickActions creates a post with quick action buttons (continue, explain, undo)
func (p *Plugin) postWithQuickActions(channelID, content, sessionID string) (string, error) {
	actions := []*model.PostAction{
		{
			Id:   "continue_" + sessionID,
			Name: "▶️ Continue",
			Type: "button",
			Integration: &model.PostActionIntegration{
				URL: p.getPluginURL() + "/api/action/continue",
				Context: map[string]interface{}{
					"session_id": sessionID,
				},
			},
		},
		{
			Id:   "explain_" + sessionID,
			Name: "💡 Explain",
			Type: "button",
			Integration: &model.PostActionIntegration{
				URL: p.getPluginURL() + "/api/action/explain",
				Context: map[string]interface{}{
					"session_id": sessionID,
				},
			},
		},
		{
			Id:    "undo_" + sessionID,
			Name:  "↩️ Undo",
			Type:  "button",
			Style: "danger",
			Integration: &model.PostActionIntegration{
				URL: p.getPluginURL() + "/api/action/undo",
				Context: map[string]interface{}{
					"session_id": sessionID,
				},
			},
		},
	}

	attachment := &model.SlackAttachment{
		Actions: actions,
	}

	post := &model.Post{
		ChannelId: channelID,
		UserId:    p.botUserID,
		Message:   content,
		Props: model.StringInterface{
			"attachments": []*model.SlackAttachment{attachment},
		},
	}

	createdPost, err := p.API.CreatePost(post)
	if err != nil {
		return "", err
	}
	return createdPost.Id, nil
}

// postCodeChange creates a post with code diff and apply/discard/view buttons
func (p *Plugin) postCodeChange(channelID, filename, diff string, changeID string) error {
	codeBlock := "```diff\n" + diff + "\n```"

	actions := []*model.PostAction{
		{
			Id:    "apply_" + changeID,
			Name:  "✅ Apply",
			Type:  "button",
			Style: "primary",
			Integration: &model.PostActionIntegration{
				URL: p.getPluginURL() + "/api/action/apply",
				Context: map[string]interface{}{
					"change_id": changeID,
					"filename":  filename,
				},
			},
		},
		{
			Id:    "discard_" + changeID,
			Name:  "❌ Discard",
			Type:  "button",
			Style: "danger",
			Integration: &model.PostActionIntegration{
				URL: p.getPluginURL() + "/api/action/discard",
				Context: map[string]interface{}{
					"change_id": changeID,
				},
			},
		},
		{
			Id:   "view_" + changeID,
			Name: "👁️ View Full File",
			Type: "button",
			Integration: &model.PostActionIntegration{
				URL: p.getPluginURL() + "/api/action/view",
				Context: map[string]interface{}{
					"change_id": changeID,
					"filename":  filename,
				},
			},
		},
	}

	attachment := &model.SlackAttachment{
		Actions: actions,
	}

	post := &model.Post{
		ChannelId: channelID,
		UserId:    p.botUserID,
		Message:   fmt.Sprintf("📝 **%s**\n\n%s", filename, codeBlock),
		Props: model.StringInterface{
			"attachments": []*model.SlackAttachment{attachment},
		},
	}

	_, err := p.API.CreatePost(post)
	return err
}

// postWithMenu creates a post with a dropdown menu of options
func (p *Plugin) postWithMenu(channelID, content string, options []ActionOption, sessionID string) error {
	menuOptions := make([]*model.PostActionOptions, len(options))
	for i, opt := range options {
		menuOptions[i] = &model.PostActionOptions{
			Text:  opt.Label,
			Value: opt.Value,
		}
	}

	action := &model.PostAction{
		Id:   "action_menu_" + sessionID,
		Name: "Actions",
		Type: "select",
		Integration: &model.PostActionIntegration{
			URL: p.getPluginURL() + "/api/action/menu",
			Context: map[string]interface{}{
				"session_id": sessionID,
			},
		},
		Options: menuOptions,
	}

	attachment := &model.SlackAttachment{
		Actions: []*model.PostAction{action},
	}

	post := &model.Post{
		ChannelId: channelID,
		UserId:    p.botUserID,
		Message:   content,
		Props: model.StringInterface{
			"attachments": []*model.SlackAttachment{attachment},
		},
	}

	_, err := p.API.CreatePost(post)
	return err
}

// updatePostWithProgress updates an existing post to show progress status
func (p *Plugin) updatePostWithProgress(postID, status string) error {
	post, err := p.API.GetPost(postID)
	if err != nil {
		return err
	}

	post.Message = "⏳ " + status
	_, err = p.API.UpdatePost(post)
	return err
}

// updatePostMessage updates an existing post's message
func (p *Plugin) updatePostMessage(postID, message string) error {
	post, err := p.API.GetPost(postID)
	if err != nil {
		return err
	}

	post.Message = message
	_, err = p.API.UpdatePost(post)
	return err
}

// getPluginURL returns the plugin's base URL for action integrations
func (p *Plugin) getPluginURL() string {
	config := p.API.GetConfig()
	siteURL := ""
	if config.ServiceSettings.SiteURL != nil {
		siteURL = *config.ServiceSettings.SiteURL
	}
	// Plugin ID from plugin.json
	const pluginID = "com.appsome.claudecode"
	return fmt.Sprintf("%s/plugins/%s", siteURL, pluginID)
}

package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

const (
	defaultMaxThreadMessages = 50
)

// ThreadContext represents formatted thread context
type ThreadContext struct {
	Content      string
	MessageCount int
	Participants []string
	ChannelName  string
	RootPostID   string
}

// GetThreadContext retrieves and formats all messages from a thread
func (p *Plugin) GetThreadContext(rootPostID, channelID string, maxMessages int) (*ThreadContext, error) {
	// Get channel info
	channel, appErr := p.API.GetChannel(channelID)
	if appErr != nil {
		return nil, fmt.Errorf("failed to get channel: %w", appErr)
	}

	// Get post thread
	postList, appErr := p.API.GetPostThread(rootPostID)
	if appErr != nil {
		return nil, fmt.Errorf("failed to get post thread: %w", appErr)
	}

	if postList == nil || len(postList.Posts) == 0 {
		return nil, fmt.Errorf("thread is empty")
	}

	// Sort posts by creation time
	posts := make([]*model.Post, 0, len(postList.Posts))
	for _, post := range postList.Posts {
		posts = append(posts, post)
	}

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].CreateAt < posts[j].CreateAt
	})

	// Limit to maxMessages if specified
	if maxMessages > 0 && len(posts) > maxMessages {
		posts = posts[len(posts)-maxMessages:]
	}

	// Track participants
	participantMap := make(map[string]bool)
	var content strings.Builder

	// Write thread header
	rootPost := postList.Posts[rootPostID]
	timestamp := time.Unix(rootPost.CreateAt/1000, 0).Format("Jan 2, 2006 15:04 MST")
	
	content.WriteString(fmt.Sprintf("Thread Context from #%s (started %s)\n\n", channel.Name, timestamp))

	// Format each post
	for _, post := range posts {
		user, appErr := p.API.GetUser(post.UserId)
		if appErr != nil {
			p.API.LogWarn("Failed to get user for post", "user_id", post.UserId, "error", appErr.Error())
			continue
		}

		participantMap[user.Username] = true

		// Format timestamp
		postTime := time.Unix(post.CreateAt/1000, 0).Format("15:04")
		
		// Write message
		content.WriteString(fmt.Sprintf("[%s at %s]:\n", user.Username, postTime))
		content.WriteString(post.Message)
		content.WriteString("\n\n")

		// Include file attachments as references
		if len(post.FileIds) > 0 {
			content.WriteString(fmt.Sprintf("  [%d file(s) attached]\n\n", len(post.FileIds)))
		}
	}

	// Collect participants
	participants := make([]string, 0, len(participantMap))
	for username := range participantMap {
		participants = append(participants, "@"+username)
	}
	sort.Strings(participants)

	return &ThreadContext{
		Content:      content.String(),
		MessageCount: len(posts),
		Participants: participants,
		ChannelName:  channel.Name,
		RootPostID:   rootPostID,
	}, nil
}

// SendThreadContext sends thread context to bridge server
func (p *Plugin) SendThreadContext(sessionID string, context *ThreadContext, action string) error {
	return p.bridgeClient.SendContext(sessionID, &ContextRequest{
		Source:   "mattermost-thread",
		ThreadID: context.RootPostID,
		Content:  context.Content,
		Action:   action,
		Metadata: &ContextMetadata{
			ChannelName:  context.ChannelName,
			RootPostID:   context.RootPostID,
			MessageCount: context.MessageCount,
			Participants: context.Participants,
		},
	})
}

package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/git"
	"github.com/radiospiel/critic/src/pkg/critic"
	"github.com/samber/lo"
)

// GetConversations returns conversations matching the given filters.
// If paths is empty, returns conversations across all files.
// If statuses is empty, returns conversations with any status.
// Conversations are filtered to the current start..end diff range.
func (s *Server) GetConversations(
	ctx context.Context,
	req *connect.Request[api.GetConversationsRequest],
) (*connect.Response[api.GetConversationsResponse], error) {
	response := depanic(func() (*api.GetConversationsResponse, error) {
		return getConversationsImpl(s, req.Msg)
	})
	return connect.NewResponse(response), nil
}

func getConversationsImpl(server *Server, req *api.GetConversationsRequest) (*api.GetConversationsResponse, error) {
	paths := req.GetPaths()
	statuses := req.GetStatuses()

	m := server.config.Messaging

	// Build status filter set (empty means accept all)
	statusSet := make(map[api.ConversationStatus]bool, len(statuses))
	for _, s := range statuses {
		statusSet[s] = true
	}

	// Collect conversations: per-file if paths given, all roots otherwise
	var criticConversations []*critic.Conversation
	if len(paths) > 0 {
		for _, path := range paths {
			convs, err := m.GetConversationsForFile(path)
			if err != nil {
				return nil, err
			}
			criticConversations = append(criticConversations, convs...)
		}
	} else {
		// No paths filter: load all root conversations then fetch full threads
		roots, err := m.GetConversations("")
		if err != nil {
			return nil, err
		}
		for _, root := range roots {
			conv, err := m.GetFullConversation(root.UUID)
			if err != nil {
				continue
			}
			criticConversations = append(criticConversations, conv)
		}
	}

	// Get current diff range for filtering and branch resolution (if session exists)
	var start, end string
	var bases []string
	if server.session != nil {
		start = server.session.GetCurrentStart()
		end = server.session.GetCurrentEnd()
		bases = server.session.GetDiffBases()
	}

	// Filter conversations to the current diff range, apply status filter,
	// and resolve branch names.
	cache := git.NewAncestryCache()
	apiConversations := make([]*api.Conversation, 0, len(criticConversations))
	for _, conv := range criticConversations {
		if start != "" {
			if !git.IsCommitInRangeCached(conv.CodeVersion, start, end, cache) {
				continue
			}
		}

		apiStatus := criticStatusToApiStatus(conv.Status)
		if len(statusSet) > 0 && !statusSet[apiStatus] {
			continue
		}

		apiConv := criticToApiConversation(conv, 0)
		if conv.CodeVersion != "" && len(bases) > 0 {
			apiConv.BranchName = git.ClosestBranchForSHACached(conv.CodeVersion, bases, cache)
		}
		apiConversations = append(apiConversations, apiConv)
	}

	return &api.GetConversationsResponse{
		Conversations: apiConversations,
	}, nil
}

func criticToApiMessage(msg critic.Message, index int) *api.Message {
	return &api.Message{
		Id:        msg.UUID,
		Author:    string(msg.Author),
		Content:   msg.Message,
		CreatedAt: msg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: msg.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		IsUnread:  msg.IsUnread,
	}
}

func criticStatusToApiStatus(status critic.ConversationStatus) api.ConversationStatus {
	switch status {
	case critic.StatusResolved:
		return api.ConversationStatus_CONVERSATION_STATUS_RESOLVED
	case critic.StatusUnresolved:
		return api.ConversationStatus_CONVERSATION_STATUS_UNRESOLVED
	case critic.StatusActive:
		return api.ConversationStatus_CONVERSATION_STATUS_ACTIVE
	case critic.StatusWaitingForResponse:
		return api.ConversationStatus_CONVERSATION_STATUS_WAITING_FOR_RESPONSE
	case critic.StatusInformal:
		return api.ConversationStatus_CONVERSATION_STATUS_INFORMAL
	case critic.StatusArchived:
		return api.ConversationStatus_CONVERSATION_STATUS_ARCHIVED
	default:
		return api.ConversationStatus_CONVERSATION_STATUS_INVALID
	}
}

func criticTypeToApiType(ct critic.ConversationType) api.ConversationType {
	switch ct {
	case critic.TypeExplanation:
		return api.ConversationType_CONVERSATION_TYPE_EXPLANATION
	case critic.TypeConversation:
		return api.ConversationType_CONVERSATION_TYPE_CONVERSATION
	default:
		return api.ConversationType_CONVERSATION_TYPE_INVALID
	}
}

func criticToApiConversation(conv *critic.Conversation, index int) *api.Conversation {
	messages := lo.Map(conv.Messages, criticToApiMessage)
	return &api.Conversation{
		Id:               conv.UUID,
		Status:           criticStatusToApiStatus(conv.Status),
		ConversationType: criticTypeToApiType(conv.ConversationType),
		FilePath:         conv.FilePath,
		LineNumber:       int32(conv.LineNumber),
		CodeVersion:      conv.CodeVersion,
		Context:          conv.Context,
		Messages:         messages,
		CreatedAt:        conv.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        conv.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

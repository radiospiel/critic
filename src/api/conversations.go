package api

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

// Type aliases for backward compatibility - the proto has been renamed from
// Comment to Conversation, but the generated code still uses the old names.
// These aliases allow using the new naming convention.
type (
	CreateConversationRequest  = CreateCommentRequest
	CreateConversationResponse = CreateCommentResponse
	GetConversationsRequest    = GetCommentsRequest
	GetConversationsResponse   = GetCommentsResponse
)

// GetConversationsSummaryRequest requests conversation summaries for all files.
type GetConversationsSummaryRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetConversationsSummaryRequest) Reset() {
	*x = GetConversationsSummaryRequest{}
}

func (x *GetConversationsSummaryRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetConversationsSummaryRequest) ProtoMessage() {}

func (x *GetConversationsSummaryRequest) ProtoReflect() protoreflect.Message {
	// Return nil since this is manually added and not in the proto descriptor
	return nil
}

// GetConversationsSummaryResponse contains conversation summaries by file path.
type GetConversationsSummaryResponse struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Summaries is a list of file conversation summaries.
	// Only includes files that have conversations.
	Summaries []*FileConversationSummary `protobuf:"bytes,1,rep,name=summaries,proto3" json:"summaries,omitempty"`
	// Error contains error details if the request failed.
	Error         *RpcError `protobuf:"bytes,15,opt,name=error,proto3" json:"error,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetConversationsSummaryResponse) Reset() {
	*x = GetConversationsSummaryResponse{}
}

func (x *GetConversationsSummaryResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetConversationsSummaryResponse) ProtoMessage() {}

func (x *GetConversationsSummaryResponse) ProtoReflect() protoreflect.Message {
	// Return nil since this is manually added and not in the proto descriptor
	return nil
}

func (x *GetConversationsSummaryResponse) GetSummaries() []*FileConversationSummary {
	if x != nil {
		return x.Summaries
	}
	return nil
}

func (x *GetConversationsSummaryResponse) GetError() *RpcError {
	if x != nil {
		return x.Error
	}
	return nil
}

// FileConversationSummary contains conversation summary for a single file.
type FileConversationSummary struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// FilePath is the git-relative path to the file.
	FilePath string `protobuf:"bytes,1,opt,name=file_path,json=filePath,proto3" json:"file_path,omitempty"`
	// TotalCount is the total number of conversations for this file.
	TotalCount int32 `protobuf:"varint,2,opt,name=total_count,json=totalCount,proto3" json:"total_count,omitempty"`
	// UnresolvedCount is the number of unresolved conversations.
	UnresolvedCount int32 `protobuf:"varint,3,opt,name=unresolved_count,json=unresolvedCount,proto3" json:"unresolved_count,omitempty"`
	// ResolvedCount is the number of resolved conversations.
	ResolvedCount int32 `protobuf:"varint,4,opt,name=resolved_count,json=resolvedCount,proto3" json:"resolved_count,omitempty"`
	// HasUnreadAIMessages indicates if there are unread AI messages.
	HasUnreadAiMessages bool          `protobuf:"varint,5,opt,name=has_unread_ai_messages,json=hasUnreadAiMessages,proto3" json:"has_unread_ai_messages,omitempty"`
	unknownFields       protoimpl.UnknownFields
	sizeCache           protoimpl.SizeCache
}

func (x *FileConversationSummary) Reset() {
	*x = FileConversationSummary{}
}

func (x *FileConversationSummary) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FileConversationSummary) ProtoMessage() {}

func (x *FileConversationSummary) ProtoReflect() protoreflect.Message {
	// Return nil since this is manually added and not in the proto descriptor
	return nil
}

func (x *FileConversationSummary) GetFilePath() string {
	if x != nil {
		return x.FilePath
	}
	return ""
}

func (x *FileConversationSummary) GetTotalCount() int32 {
	if x != nil {
		return x.TotalCount
	}
	return 0
}

func (x *FileConversationSummary) GetUnresolvedCount() int32 {
	if x != nil {
		return x.UnresolvedCount
	}
	return 0
}

func (x *FileConversationSummary) GetResolvedCount() int32 {
	if x != nil {
		return x.ResolvedCount
	}
	return 0
}

func (x *FileConversationSummary) GetHasUnreadAiMessages() bool {
	if x != nil {
		return x.HasUnreadAiMessages
	}
	return false
}

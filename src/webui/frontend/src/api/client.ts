import { createConnectTransport } from '@connectrpc/connect-web'
import { createPromiseClient } from '@connectrpc/connect'
import { CriticService } from '../gen/critic_connect'

// Create a transport for the Connect protocol
const transport = createConnectTransport({
  baseUrl: window.location.origin,
})

// Create the client
export const criticClient = createPromiseClient(CriticService, transport)

// Types for comments (matching the backend JSON response)
export interface CommentMessage {
  id: string
  author: string
  content: string
  createdAt: string
  updatedAt: string
  isUnread: boolean
}

export interface CommentConversation {
  id: string
  status: string
  filePath: string
  lineNumber: number
  codeVersion: string
  context: string
  messages: CommentMessage[]
  createdAt: string
  updatedAt: string
}

export interface GetCommentsResponse {
  conversations: CommentConversation[]
  error?: string
}

// Fetch comments for a file using the REST endpoint
export async function getComments(filePath: string): Promise<GetCommentsResponse> {
  const response = await fetch(`/api/comments?path=${encodeURIComponent(filePath)}`)
  return response.json()
}

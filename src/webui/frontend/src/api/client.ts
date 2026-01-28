import { createConnectTransport } from '@connectrpc/connect-web'
import { createPromiseClient } from '@connectrpc/connect'
import { CriticService } from '../gen/critic_connect'

// Create a transport for the Connect protocol
const transport = createConnectTransport({
  baseUrl: window.location.origin,
})

// Create the client
export const criticClient = createPromiseClient(CriticService, transport)

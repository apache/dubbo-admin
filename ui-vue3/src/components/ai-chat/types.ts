/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

export interface Session {
  session_id: string
  created_at: string
  updated_at: string
  message_count: number
  status: string
}

export interface ChatMessage {
  id: string
  content: string
  role: 'user' | 'assistant'
  timestamp: number
  type?: 'normal' | 'error' | 'partial_error'
}

export interface ChatResponse {
  data: {
    session_id: string
    messages: ChatMessage[]
  }
}

/**
 * Normalized transport contract consumed by the reusable drawer.
 * Backend-specific URLs, headers, and payload formats belong in an adapter.
 */
export interface ChatService<TContext = unknown> {
  createSession(): Promise<string>
  getSessions?(): Promise<Session[]>
  getSessionInfo?(sessionId: string): Promise<ChatResponse>
  deleteSession?(sessionId: string): Promise<void>
  sendChatMessage(
    message: string,
    sessionId?: string,
    context?: TContext
  ): Promise<ReadableStream<Uint8Array>>
}

export interface ChatSuggestion {
  icon: string
  iconColor: string
  title: string
  content: string
}

export interface AIChatLabels {
  title: string
  welcomeMessage: string
  welcomeTagline: string
  inputTokens: string
  outputTokens: string
  totalTokens: string
  newChat: string
  history: string
  clearHistory: string
  placeholder: string
  thinking: string
  errorTitle: string
  retry: string
  historyTitle: string
  emptyHistory: string
  session: string
  messageCount: string
  newSession: string
  delete: string
  createSessionFailed: string
  createSessionUnavailable: string
  sessionDeleted: string
  deleteSessionFailed: string
  noRetryMessage: string
  waitForResponse: string
  unknownStreamError: string
  parseStreamError: string
  requestFailed: string
  networkError: string
  serverError: string
  processingError: string
  historyCleared: string
  newChatCreated: string
  loadSessionFailed: string
}

export const DEFAULT_AI_CHAT_LABELS: AIChatLabels = {
  title: 'AI Assistant',
  welcomeMessage: 'Ask a question to start a conversation.',
  welcomeTagline: 'Suggested questions',
  inputTokens: 'Input tokens',
  outputTokens: 'Output tokens',
  totalTokens: 'Total tokens',
  newChat: 'New chat',
  history: 'History',
  clearHistory: 'Clear history',
  placeholder: 'Ask a question. Press Shift + Enter for a new line.',
  thinking: 'Thinking...',
  errorTitle: 'Something went wrong',
  retry: 'Retry',
  historyTitle: 'Conversation history',
  emptyHistory: 'No conversation history',
  session: 'Session',
  messageCount: 'Messages',
  newSession: 'New session',
  delete: 'Delete',
  createSessionFailed: 'Failed to create a session',
  createSessionUnavailable: 'Unable to create a session. Please try again later.',
  sessionDeleted: 'Session deleted',
  deleteSessionFailed: 'Failed to delete the session',
  noRetryMessage: 'There is no message to retry',
  waitForResponse: 'Wait for the current response to finish',
  unknownStreamError: 'An unknown error occurred while processing the response',
  parseStreamError: 'Failed to parse the server response',
  requestFailed: 'Failed to send the message. Please try again later.',
  networkError: 'Network connection failed. Check your connection and retry.',
  serverError: 'The server returned an error. Please try again later.',
  processingError: 'Sorry, an error occurred while processing the message.',
  historyCleared: 'History cleared',
  newChatCreated: 'New chat created',
  loadSessionFailed: 'Failed to load the session'
}

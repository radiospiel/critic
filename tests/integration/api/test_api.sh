#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/testsh.inc"

CRITIC_BIN="${CRITIC_BIN:-$SCRIPT_DIR/critic}"
PORT=0  # will be assigned dynamically
BASE_URL="" # set after server starts

# -- helpers ------------------------------------------------------------------

# Create a temporary git repo for testing
setup_git_repo() {
  WORK_DIR=$(mktemp -d)
  cd "$WORK_DIR"
  git init --quiet
  git config user.name "Test User"
  git config user.email "test@example.com"
  git config commit.gpgsign false

  # Create and commit a file so we have a valid HEAD
  echo "package main" > main.go
  git add main.go
  git commit --quiet -m "initial commit"
}

# Find a free port
find_free_port() {
  python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()'
}

# Start the critic HTTP server in the background
start_server() {
  PORT=$(find_free_port)
  BASE_URL="http://localhost:${PORT}"

  "$CRITIC_BIN" httpd --port="$PORT" &>/dev/null &
  SERVER_PID=$!

  # Wait for server to be ready
  local retries=0
  while ! curl -s "$BASE_URL" >/dev/null 2>&1; do
    retries=$((retries + 1))
    if (( retries > 50 )); then
      echo "Server failed to start within timeout"
      return 1
    fi
    sleep 0.1
  done
}

# Stop the critic HTTP server
stop_server() {
  if [[ -n "${SERVER_PID:-}" ]]; then
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
    unset SERVER_PID
  fi
}

# Make a Connect-RPC call. Usage: rpc <method> <json_body>
# Returns the JSON response body.
rpc() {
  local method="$1"
  local body="${2:-\{\}}"
  curl -s \
    -X POST \
    -H "Content-Type: application/json" \
    -d "$body" \
    "${BASE_URL}/critic.v1.CriticService/${method}"
}

# -- lifecycle ----------------------------------------------------------------

setup() {
  setup_git_repo
  start_server
}

teardown() {
  stop_server
  if [[ -n "${WORK_DIR:-}" ]]; then
    rm -rf "$WORK_DIR"
    unset WORK_DIR
  fi
}

# -- tests --------------------------------------------------------------------

test_get_last_change_returns_timestamp() {
  local response
  response=$(rpc GetLastChange '{}')

  # Response should contain mtimeMsecs (Connect-RPC uses camelCase JSON)
  local mtime
  mtime=$(echo "$response" | jq -r '.mtimeMsecs // empty')
  assert_neq "" "$mtime" "GetLastChange should return mtimeMsecs"

  # Timestamp should be a reasonable value (> 0)
  assert_true "[[ $mtime -gt 0 ]]" "mtimeMsecs should be greater than 0"
}

test_create_conversation_and_verify_last_change() {
  # Record initial last change timestamp
  local initial_response
  initial_response=$(rpc GetLastChange '{}')
  local initial_mtime
  initial_mtime=$(echo "$initial_response" | jq -r '.mtimeMsecs')

  # Small delay to ensure timestamp can change
  sleep 1.1

  # Create a conversation (post a comment)
  local create_response
  create_response=$(rpc CreateConversation '{
    "newFile": "main.go",
    "newLine": 1,
    "comment": "This needs a doc comment"
  }')

  local success
  success=$(echo "$create_response" | jq -r '.success')
  assert_eq "true" "$success" "CreateConversation should succeed"

  # Wait for database watcher to pick up the change (polls every 1s)
  sleep 1.5

  # GetLastChange should now return a newer timestamp
  local updated_response
  updated_response=$(rpc GetLastChange '{}')
  local updated_mtime
  updated_mtime=$(echo "$updated_response" | jq -r '.mtimeMsecs')

  assert_true "[[ $updated_mtime -gt $initial_mtime ]]" \
    "GetLastChange mtime should increase after creating a conversation (initial=$initial_mtime, updated=$updated_mtime)"
}

test_conversation_is_in_get_conversations() {
  # Create a conversation
  rpc CreateConversation '{
    "newFile": "main.go",
    "newLine": 1,
    "comment": "Please add error handling here"
  }' >/dev/null

  # Retrieve conversations
  local response
  response=$(rpc GetConversations '{"paths": ["main.go"]}')

  # Verify the conversation exists
  local count
  count=$(echo "$response" | jq '.conversations | length')
  assert_eq "1" "$count" "Should have exactly 1 conversation"

  # Verify the comment content
  local comment
  comment=$(echo "$response" | jq -r '.conversations[0].messages[0].content')
  assert_eq "Please add error handling here" "$comment" "Comment content should match"

  # Verify the author
  local author
  author=$(echo "$response" | jq -r '.conversations[0].messages[0].author')
  assert_eq "human" "$author" "Author should be human"
}

test_reply_and_verify_last_change() {
  # Create a conversation
  rpc CreateConversation '{
    "newFile": "main.go",
    "newLine": 1,
    "comment": "Why is this public?"
  }' >/dev/null

  # Get the conversation ID
  local get_response
  get_response=$(rpc GetConversations '{"paths": ["main.go"]}')
  local conv_id
  conv_id=$(echo "$get_response" | jq -r '.conversations[0].id')
  assert_neq "" "$conv_id" "Conversation should have an ID"

  # Record timestamp before reply
  local before_response
  before_response=$(rpc GetLastChange '{}')
  local before_mtime
  before_mtime=$(echo "$before_response" | jq -r '.mtimeMsecs')

  # Small delay to ensure timestamp can change
  sleep 1.1

  # Add a reply
  local reply_response
  reply_response=$(rpc ReplyToConversation "{
    \"conversationId\": \"$conv_id\",
    \"message\": \"Good point, making it private\"
  }")

  local reply_success
  reply_success=$(echo "$reply_response" | jq -r '.success')
  assert_eq "true" "$reply_success" "ReplyToConversation should succeed"

  # Wait for database watcher to pick up the change
  sleep 1.5

  # GetLastChange should return a newer timestamp
  local after_response
  after_response=$(rpc GetLastChange '{}')
  local after_mtime
  after_mtime=$(echo "$after_response" | jq -r '.mtimeMsecs')

  assert_true "[[ $after_mtime -gt $before_mtime ]]" \
    "GetLastChange mtime should increase after reply (before=$before_mtime, after=$after_mtime)"
}

test_reply_is_in_conversation() {
  # Create a conversation
  rpc CreateConversation '{
    "newFile": "main.go",
    "newLine": 1,
    "comment": "Consider using a constant"
  }' >/dev/null

  # Get the conversation ID
  local get_response
  get_response=$(rpc GetConversations '{"paths": ["main.go"]}')
  local conv_id
  conv_id=$(echo "$get_response" | jq -r '.conversations[0].id')

  # Add a reply
  rpc ReplyToConversation "{
    \"conversationId\": \"$conv_id\",
    \"message\": \"Done, moved to constants.go\"
  }" >/dev/null

  # Fetch the conversation again
  local updated_response
  updated_response=$(rpc GetConversations '{"paths": ["main.go"]}')

  # Verify the conversation now has 2 messages
  local msg_count
  msg_count=$(echo "$updated_response" | jq '.conversations[0].messages | length')
  assert_eq "2" "$msg_count" "Conversation should have 2 messages (comment + reply)"

  # Verify the original comment
  local original_comment
  original_comment=$(echo "$updated_response" | jq -r '.conversations[0].messages[0].content')
  assert_eq "Consider using a constant" "$original_comment" "First message should be the original comment"

  # Verify the reply
  local reply_content
  reply_content=$(echo "$updated_response" | jq -r '.conversations[0].messages[1].content')
  assert_eq "Done, moved to constants.go" "$reply_content" "Second message should be the reply"
}

# -- run ----------------------------------------------------------------------

run_tests

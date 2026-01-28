# Task: Replace webui with React app with file list

**Started:** 2026-01-27 14:42:52
**Ended:** 2026-01-28 06:25:00
**Strategy:** Feature
**Status:** Completed
**Complexity:** Medium
**Used Models:** Opus

## Objective
Drop existing webui source code (templates, static files) and create a new React app with:
- Left panel (30%): File list using GetDiffs proto API
- Right panel (70%): Empty (for future file rendering)

## Progress
- [x] Remove existing templates and static files
- [x] Create React app structure (package.json, configs)
- [x] Create main App component with layout
- [x] Create FileList component
- [x] Update Go backend to serve React app
- [x] Set up protobuf TypeScript generation with buf
- [x] Generate TypeScript types from proto (critic_pb.ts, critic_connect.ts)
- [x] Add Connect RPC handler to webui server
- [x] Update FileList to use GetDiffs proto API
- [x] Test the implementation (Go code compiles, React builds)

## Obstacles
- Initial npm package versions for @connectrpc were wrong (2.0.0 doesn't exist for protoc-gen-connect-es), fixed by using 1.x versions

## Outcome
Replaced the HTMX-based webui with a new React app using proto-based API:
- Created `src/webui/frontend/` with Vite + React + TypeScript
- Layout: 30% left sidebar (FileList), 70% right content area
- Set up buf for TypeScript code generation from proto files
- FileList component uses `criticClient.getDiffs()` from generated Connect client
- Added Connect RPC handler to webui server (connect_api.go)
- File status displayed with color-coded badges (M=Modified, A=Added, D=Deleted, R=Renamed)
- Go backend serves React build from embedded `dist/` directory

## Insights
- Vite provides a fast and simple build setup for React apps
- Go's embed directive works well for embedding the built React app
- Using buf for proto code generation is straightforward with proper config
- Connect RPC provides a clean client API with full TypeScript type safety

# Task: Replace webui with React app with file list

**Started:** 2026-01-27 14:42:52
**Ended:** 2026-01-28 09:30:00
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
- [x] Refactor webui to be minimal (only DistFS and WebSocketHandler)
- [x] Integrate webui into API server (no separate webui CLI command)
- [x] Add development mode with Vite hot reload (`--dev` flag)
- [x] Test the implementation (Go code compiles, React builds)

## Obstacles
- Initial npm package versions for @connectrpc were wrong (2.0.0 doesn't exist for protoc-gen-connect-es), fixed by using 1.x versions
- Route pattern conflict in Go 1.22+ ServeMux between `GET /` and `/critic.v1.CriticService/`, fixed by removing `GET` method prefix

## Outcome
Replaced the HTMX-based webui with a new React app using proto-based API:
- Created `src/webui/frontend/` with Vite + React + TypeScript
- Layout: 30% left sidebar (FileList), 70% right content area
- Set up buf for TypeScript code generation from proto files
- FileList component uses `criticClient.getDiffs()` from generated Connect client
- File status displayed with color-coded badges (M=Modified, A=Added, D=Deleted, R=Renamed)
- Go backend serves React build from embedded `dist/` directory
- Webui package is now minimal: only exports DistFS() and WebSocketHandler()
- Web UI is now part of the API server (`critic api` command)
- Development mode: `critic api --dev` proxies to Vite dev server for hot reload

## Development Workflow
To develop the frontend with hot reload:
1. Start Vite dev server: `cd src/webui/frontend && npm run dev`
2. Start API server in dev mode: `critic api --dev`
3. Open http://localhost:65432 in browser
4. Changes to frontend code will hot reload automatically

## Insights
- Vite provides a fast and simple build setup for React apps
- Go's embed directive works well for embedding the built React app
- Using buf for proto code generation is straightforward with proper config
- Connect RPC provides a clean client API with full TypeScript type safety
- Go's httputil.ReverseProxy makes it easy to proxy to Vite dev server

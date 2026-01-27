# Task: Print grpcurl command on API server startup

**Started:** 2026-01-27 06:41:12
**Ended:** 2026-01-27 06:43:00
**Strategy:** Feature (TDD)
**Status:** Completed
**Complexity:** Simple
**Used Models:** Opus

## Objective
When starting "critic api", print the full grpcurl command to fetch a GetLastChangeRequest.

## Progress
- [x] Explored codebase to understand API server structure
- [x] Found server implementation in src/api/server/server.go
- [x] Add grpcurl command print to server startup
- [x] Add curl command print (user requested)
- [x] Add handler at root path for grpcurl compatibility
- [x] Test the output

## Obstacles
- **Issue:** grpcurl expects gRPC handlers at root path, but server mounted at `/api` prefix
  **Resolution:** Added handler at both `/api` prefix and root path

## Outcome
Server now prints both grpcurl and curl commands on startup:
```
API server listening on :65432

Test with grpcurl:
  grpcurl -plaintext -import-path src/api/proto -proto critic.proto localhost:65432 critic.v1.CriticService/GetLastChange

Test with curl:
  curl -X POST http://localhost:65432/api/critic.v1.CriticService/GetLastChange -H 'Content-Type: application/json' -d '{}'
```

## Insights
- Connect servers support gRPC protocol but path prefixes break grpcurl compatibility
- Solution: mount handlers at both custom prefix and root path

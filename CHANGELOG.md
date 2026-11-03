# Changelog

## Unreleased

### Fixed

- `Track` and `PageView` generated a new session id and visitor id per call, so every server-side
  event counted as a new visitor and a new visit. When `IPAddress` or `UserAgent` is given, as the
  HTTP middleware does, the visitor id is now derived from them per UTC day and the session id per
  30-minute UTC window, cookieless and identical to the other Entrolytics server SDKs. An explicit
  `SessionID` or `VisitorID` still wins; without visitor context new ids per call are kept.
- The module did not compile: `client.go`, a leftover of an earlier client that sent the legacy
  payload shape, declared `package client` in the `entrolytics` package directory, so `go build`,
  `go vet` and `go get` failed. It is removed. Its only feature the package lacked, batch
  submission, is now `Batch` / `BatchWithContext`, which send up to 100 custom events in one
  request to `/collect/batch` in the canonical payload shape; the events of a batch must share one
  `IPAddress` and `UserAgent`, since the API resolves geo data and screens bots per request.

# Changelog

## Unreleased

### Fixed

- `Track` and `PageView` generated a new session id and visitor id per call, so every server-side
  event counted as a new visitor and a new visit. When `IPAddress` or `UserAgent` is given, as the
  HTTP middleware does, the visitor id is now derived from them per UTC day and the session id per
  30-minute UTC window, cookieless and identical to the other Entrolytics server SDKs. An explicit
  `SessionID` or `VisitorID` still wins; without visitor context new ids per call are kept.

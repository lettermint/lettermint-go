# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## v2.0.0 - 2026-05-11

### What's Changed

* feat: add Lettermint Team API support by @bjarn in https://github.com/lettermint/lettermint-go/pull/14

**Full Changelog**: https://github.com/lettermint/lettermint-go/compare/v1.1.1...v2.0.0

## v1.1.1 - 2026-05-05

### What's Changed

* ci: bump dependabot/fetch-metadata from 3.0.0 to 3.1.0 by @dependabot[bot] in https://github.com/lettermint/lettermint-go/pull/12
* feat: add request body size limit for webhook verification by @bjarn in https://github.com/lettermint/lettermint-go/pull/13

**Full Changelog**: https://github.com/lettermint/lettermint-go/compare/v1.1.0...v1.1.1

## v1.0.1 - 2025-12-23

### What's Changed

* feat: enhance User-Agent header with runtime version by @bjarn in https://github.com/lettermint/lettermint-go/pull/1

### New Contributors

* @bjarn made their first contribution in https://github.com/lettermint/lettermint-go/pull/1

**Full Changelog**: https://github.com/lettermint/lettermint-go/compare/v1.0.0...v1.0.1

## v1.0.0 - 2025-12-23

**Full Changelog**: https://github.com/lettermint/lettermint-go/commits/v1.0.0

## [Unreleased]

## [1.0.0] - TBD

### Added

- Initial release of the Lettermint Go SDK
- Email sending with fluent builder interface
- Support for HTML and plain text emails
- CC, BCC, and Reply-To support
- File attachments with inline embedding support
- Custom headers, metadata, and tags
- Idempotency key support
- Webhook signature verification with HMAC-SHA256
- Timestamp tolerance validation for webhooks
- Comprehensive error types with `errors.Is()` / `errors.As()` support
- Functional options for client configuration
- Context support for request cancellation and timeouts

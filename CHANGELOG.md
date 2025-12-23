# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

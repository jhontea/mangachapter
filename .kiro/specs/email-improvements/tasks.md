# Implementation Plan: Email Notification Improvements

## Overview

This plan breaks down the 6-week email notification improvements feature into discrete implementation tasks. The work follows a logical sequence: core infrastructure (providers, rate limiting), template system, retry logic, attachments, testing/documentation, and polish/validation.

## Tasks

- [ ] 1. Week 1: Core Infrastructure
  - [ ] 1.1 Create provider package structure
    - Create `internal/notifier/provider/provider.go` with `EmailProvider` enum and provider interface
    - _Requirements: 1.1, 1.3_
  
  - [ ] 1.2 Implement provider-specific configurations
    - Create `internal/notifier/provider/gmail.go` with Gmail-specific settings (port 587, App Password requirement)
    - Create `internal/notifier/provider/outlook.go` with Outlook-specific settings
    - Create `internal/notifier/provider/proton.go` with Proton Mail settings
    - Create `internal/notifier/provider/zoho.go` with Zoho-specific settings
    - Create `internal/notifier/provider/generic.go` for generic SMTP
    - Each provider file defines struct with Host, Port, UseTLS, UseStartTLS, RateLimitPerDay, RateLimitPerMinute
    - _Requirements: 1.1, 1.3, 6.2, 6.3, 6.4_
  
  - [ ] 1.3 Implement rate limiter with provider-specific quotas
    - Create `internal/notifier/rate_limit.go`
    - Implement `RateLimiter` struct with `SentToday`, `SentLastMinute`, `LastReset`, `Quota`
    - Implement `CanSend() bool` method to check quota availability
    - Implement `RecordSent() error` method to track sent emails
    - Implement `ResetIfNecessary() error` method for quota resets
    - Use sliding window for per-minute tracking
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_

- [ ] 2. Week 1: Refactor Email Notifier
  - [ ] 2.1 Update config structure for new fields
    - Modify `internal/config/config.go` EmailConfig struct:
      - Add `Provider EmailProvider` field
      - Add `EnableHTML bool` field (default: true)
      - Add `TemplateDir string` field
      - Add `Attachments []string` field
      - Add `MaxAttachmentSize int64` field (default: 25MB)
      - Add `RateLimit RateLimitConfig` struct with PerDay and PerMinute
      - Add `Retry RetryConfig` struct with MaxAttempts, InitialBackoff, MaxBackoff
    - Add `RateLimitConfig` and `RetryConfig` nested structs
    - Update `validate()` to validate new provider-specific fields
    - Update `applyEnvOverrides()` to handle new config fields
    - _Requirements: 8.5, 10.2_

- [ ] 3. Week 1: Rate Limiter Integration
  - [ ] 3.1 Integrate rate limiter with EmailNotifier
    - Add `RateLimiter *RateLimiter` field to EmailNotifier
    - Update `NewEmail()` to accept rate limiter
    - Modify `SendNewChapter()` to check `CanSend()` before attempting send
    - Add `RecordSent()` call after successful sends
    - Log warnings when approaching limits (80%, 90%, 95%)
    - _Requirements: 6.1, 6.5_

- [ ] 4. Week 2: Template System
  - [ ] 4.1 Create template rendering module
    - Create `internal/notifier/template.go`
    - Implement `RenderTemplate()` function that takes template path, type (html/text), and data
    - Support Go template syntax for dynamic content
    - Implement graceful fallback to built-in templates if custom template missing
    - _Requirements: 3.1, 3.5, 7.1, 7.2, 7.3, 7.4, 7.5_
  
  - [ ] 4.2 Create default HTML template
    - Include CSS styling for headings, links, code blocks, separators
    - Render chapter URL as clickable `<a>` tag
    - Generate both text/plain and text/html parts for multipart emails
    - _Requirements: 3.2, 3.3, 3.4_

- [ ] 5. Week 2: Template Integration
  - [ ] 5.1 Update EmailNotifier to use templates
    - Modify `SendNewChapter()` to generate multipart message with HTML and plain text
    - Load templates from `TemplateDir` if configured, else use defaults
    - Validate that all required fields have values (use defaults if empty)
    - _Requirements: 3.1, 3.2, 3.5, 7.1, 7.2, 7.3, 7.5_

- [ ] 6. Week 3: Retry Logic
  - [ ] 6.1 Implement retry handler with exponential backoff
    - Create `internal/notifier/retry.go` with error types:
      - `SendError` with Retryable, RetryCount, MaxRetries fields
      - `RateLimitError` for rate limit violations
      - `TemplateError` for template rendering issues
    - Implement `SendWithRetry()` method on EmailNotifier
    - Calculate exponential backoff: 1s, 2s, 4s (double each attempt)
    - _Requirements: 5.1, 5.2, 5.3, 5.4_

- [ ] 7. Week 3: Retry Integration
  - [ ] 7.1 Integrate retry logic with notification flow
    - Call `SendWithRetry()` instead of `sendEmail()` directly
    - Handle retryable vs non-retryable errors correctly
    - Log errors before each retry attempt
    - _Requirements: 5.1, 5.3, 5.4, 10.1_

- [ ] 8. Week 4: Attachment Handling
  - [ ] 8.1 Create attachment management module
    - Create `internal/notifier/attachment.go`
    - Implement `Attachment` struct with Path, Filename, MimeType, Data, Size, Encoded fields
    - Implement `Load()` method to read file content
    - Implement `Validate()` method to check file size (max 25MB)
    - Implement `EncodeBase64()` method for SMTP transmission
    - _Requirements: 4.1, 4.2, 4.3, 4.4_

- [ ] 9. Week 4: Attachment Integration
  - [ ] 9.1 Update EmailNotifier to support attachments
    - Add Attachments field to EmailNotifier
    - Load and encode all configured attachments before sending
    - Skip attachments that fail validation (size > 25MB or unreadable)
    - Include attachments in multipart email message
    - _Requirements: 4.1, 4.2, 4.3, 4.4_

- [ ] 10. Week 5: Testing Infrastructure
  - [ ] 10.1 Create mock SMTP server
    - Create `internal/notifier/test/mock_smtp.go`
    - Implement `MockSMTPServer` struct with Addr, Messages, Errors, Port fields
    - Implement `Start() error`, `Stop() error`, `GetMessageCount() int`, `GetMessages() []smtp.Message` methods
    - Support capturing multiple messages for testing
    - _Requirements: 9.1, 9.4_

- [ ] 11. Week 5: Unit Tests
  - [ ] 10.2 Write unit tests for rate limiter
    - Test `CanSend()` with various quota states
    - Test `RecordSent()` increments counters correctly
    - Test `ResetIfNecessary()` resets at appropriate times
    - Test provider-specific quotas (Gmail 500/day, Outlook 30/min)
    - _Requirements: 6.1, 6.2, 6.3, 6.4*
  
  - [ ]* 10.3 Write unit tests for retry logic
    - Test retry on transient failures (timeout, rate limit)
    - Test exponential backoff timing
    - Test max retries prevents infinite loops
    - Test non-retryable errors don't trigger retries
    - _Requirements: 5.1, 5.2, 5.3*

- [ ] 12. Week 5: Testing Infrastructure (continued)
  - [ ] 10.4 Write unit tests for template rendering
    - Test default template rendering with sample data
    - Test custom template loading from directory
    - Test Go template variable substitution
    - Test fallback when template file missing
    - _Requirements: 3.5, 7.1, 7.2, 7.3, 7.4*

- [ ] 13. Week 5: Testing Infrastructure (continued)
  - [ ] 10.5 Write unit tests for attachment handling
    - Test valid attachment loads and encodes correctly
    - Test oversized files are skipped
    - Test non-existent files handled gracefully
    - Test Base64 encoding matches expected format
    - _Requirements: 4.1, 4.2, 4.3, 4.4*

- [ ] 14. Week 5: Provider Documentation
  - [ ] 10.6 Create Gmail setup documentation
    - Create `docs/SETUP_GMAIL.md`
    - Include steps for enabling 2FA
    - Document App Password generation process
    - Provide troubleshooting tips for common issues
    - _Requirements: 2.1*
  
  - [ ] 10.7 Create Outlook setup documentation
    - Create `docs/SETUP_OUTLOOK.md`
    - Document App Password creation for Outlook 365
    - Include SMTP settings (host, port, TLS)
    - Provide troubleshooting tips
    - _Requirements: 2.2*
  
  - [ ] 10.8 Create Proton Mail setup documentation
    - Create `docs/SETUP_PROTON.md`
    - Document Proton Mail Bridge or SMTP forwarding setup
    - Include provider-specific requirements
    - _Requirements: 2.3*
  
  - [ ] 10.9 Create Zoho Mail setup documentation
    - Create `docs/SETUP_ZOHO.md`
    - Document App Password generation and SMTP access enablement
    - Include free account limits (100/day)
    - _Requirements: 2.4*
  
  - [ ] 10.10 Create custom SMTP documentation
    - Create `docs/SETUP_CUSTOM.md`
    - Document generic SMTP configuration
    - Include common troubleshooting tips for connection issues
    - _Requirements: 2.5*

- [ ] 15. Week 6: Database Integration
  - [ ] 11.1 Update database schema for failed notifications
    - Modify `internal/storage/models.go` to add `FailedNotification` struct:
      - ID, MangaID, Chapter, ChapterURL, ErrorMsg, FailedAt, RetryCount
    - Create migration script or SQL to add table
    - _Requirements: 10.3_

- [ ] 16. Week 6: CLI Status Updates
  - [ ] 11.2 Update check.go to show notification status
    - Modify `printResults()` to include notification status (OK, SENT, FAILED, SKIPPED)
    - Add notification error details when available
    - _Requirements: 10.4*

- [ ] 17. Week 6: Full Integration
  - [ ] 11.3 Wire everything together in app.go
    - Update `initNotifier()` to:
      - Load provider config based on email.provider
      - Initialize rate limiter with provider-specific quotas
      - Configure template directory from config
      - Set up attachment paths from config
      - Configure retry settings
    - Ensure graceful degradation when email config is invalid
    - _Requirements: 8.2, 10.2*

- [ ] 18. Week 6: Validation and Testing
  - [ ] 11.4 Write comprehensive integration tests
    - Test full notification flow end-to-end
    - Test provider-specific configurations
    - Test retry scenarios with mock SMTP
    - Test attachment loading and encoding
    - _Requirements: 9.3*

- [ ] 19. Week 6: Polish
  - [ ] 11.5 Security review
    - Verify passwords are never logged in plain text
    - Add masking for password fields in debug logs
    - Verify TLS configuration for ports 587 (STARTTLS) and 465 (implicit TLS)
    - _Requirements: 8.1, 8.3*

- [ ] 20. Final Checkpoint - Ensure all tests pass
  - Run `go test ./...` and fix any failures
  - Run `go build ./cmd/manga` to verify compilation
  - Ensure all unit tests pass
  - Ensure all integration tests pass if run
  - _Requirements: All_

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP. Core implementation tasks (without *) must be completed.
- Provider-specific files use the enum pattern for easy addition of new providers
- Rate limiter uses sliding window for accurate per-minute tracking
- Template system supports Go's built-in html/template for security
- Retry logic uses exponential backoff (1s, 2s, 4s) capped at 30s
- Attachments >25MB are skipped without compression to avoid email provider size limits
- Failed notifications are logged to database for later review
- CLI commands report status without exiting on email failures

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1"] },
    { "id": 1, "tasks": ["1.2", "2.1"] },
    { "id": 2, "tasks": ["1.3", "3.1"] },
    { "id": 3, "tasks": ["4.1", "4.2"] },
    { "id": 4, "tasks": ["5.1"] },
    { "id": 5, "tasks": ["6.1"] },
    { "id": 6, "tasks": ["7.1", "8.1"] },
    { "id": 7, "tasks": ["9.1", "10.1", "10.6", "10.7", "10.8", "10.9", "10.10"] },
    { "id": 8, "tasks": ["10.2", "10.3", "10.4", "10.5", "11.1"] },
    { "id": 9, "tasks": ["11.2", "11.3", "11.4"] },
    { "id": 10, "tasks": ["11.5"] },
    { "id": 11, "tasks": ["12.1"] }
  ]
}
```

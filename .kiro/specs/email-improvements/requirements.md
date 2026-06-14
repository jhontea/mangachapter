# Requirements Document: Email Notification Improvements

## Introduction

This document outlines requirements for improving the email notification system in the Manga Chapter Notifier application. The improvements include support for multiple email providers with detailed setup instructions, HTML email templates, retry logic with exponential backoff, rate limiting, and additional features to enhance reliability and user experience.

## Glossary

- **Email Notifier**: The component responsible for sending chapter notifications via email
- **SMTP**: Simple Mail Transfer Protocol - standard protocol for sending emails
- **TLS**: Transport Layer Security - encryption protocol for secure SMTP connections
- **STARTTLS**: Command to upgrade a plain connection to an encrypted one
- **App Password**: A special password generated for applications that need to access Gmail accounts
- **HTML Template**: A formatted email template using HTML/CSS for better presentation
- **Retry Logic**: Automatic reattempt of failed email sends with increasing delays
- **Rate Limiting**: Controlling the frequency of email sends to avoid SMTP throttling
- **SMTP Provider**: Email service provider offering SMTP relay (Gmail, Outlook, Proton, Zoho, etc.)

## Requirements

### Requirement 1: Multiple Email Provider Support

**User Story:** As a user, I want to use different email providers with the email notifier, so that I can choose the service that best fits my needs.

#### Acceptance Criteria

1. WHERE an email provider is selected, THE Email Notifier SHALL support at least the following providers: Gmail, Outlook, Proton Mail, Zoho Mail, and generic SMTP
2. WHEN the application loads configuration, THE System SHALL validate that the configured provider's SMTP settings are compatible
3. THE Email Notifier SHALL document provider-specific requirements including SMTP host, port, and authentication method for each supported provider

### Requirement 2: Provider-Specific Setup Instructions

**User Story:** As a user, I want clear setup instructions for each email provider, so that I can configure the notifier correctly without external research.

#### Acceptance Criteria

1. WHEN configuring Gmail, THE Documentation SHALL include steps for enabling 2FA and generating an App Password
2. WHEN configuring Outlook, THE Documentation SHALL include steps for creating an App Password for Outlook 365 accounts
3. WHEN configuring Proton Mail, THE Documentation SHALL include steps for setting up Proton Mail Bridge or SMTP forwarding
4. WHEN configuring Zoho Mail, THE Documentation SHALL include steps for generating an App Password and enabling SMTP access
5. WHERE a custom SMTP server is used, THE Documentation SHALL include common troubleshooting tips for connection issues

### Requirement 3: HTML Email Templates

**User Story:** As a user, I want to receive HTML-formatted email notifications, so that I can get a better visual experience with proper formatting and styling.

#### Acceptance Criteria

1. WHERE email notifications are enabled, THE System SHALL support both plain text and HTML email templates
2. WHEN an HTML template is used, THE Email Notifier SHALL generate a multipart email with both text/plain and text/html parts
3. THE HTML template SHALL include styling for common elements: headings, links, code blocks, and separators
4. FOR ALL manga chapter notifications, THE System SHALL render the chapter URL as a clickable link in HTML emails
5. WHERE a custom HTML template file is provided, THE System SHALL load and use it instead of the default template

### Requirement 4: Email Attachments Support

**User Story:** As a user, I want to include chapter images or additional files as attachments, so that I can receive context directly in the email.

#### Acceptance Criteria

1. WHERE attachments are configured, THE System SHALL support adding file attachments to email notifications
2. WHEN an attachment is added, THE Email Notifier SHALL encode it in Base64 format for SMTP transmission
3. WHERE an attachment file exceeds 25MB, THEN THE System SHALL skip it immediately without attempting compression or resizing
4. IF an attachment file cannot be read or exceeds the size limit, THEN THE System SHALL skip that attachment regardless of whether warning logging succeeds

### Requirement 5: Retry Logic for Failed Emails

**User Story:** As a user, I want failed email sends to be retried automatically, so that transient network issues don't cause me to miss important notifications.

#### Acceptance Criteria

1. WHEN an email send fails due to a transient error (connection timeout, rate limit), THE Email Notifier SHALL retry the send up to 3 times
2. BETWEEN retries, THE System SHALL wait with exponential backoff (1s, 2s, 4s)
3. IF all retry attempts fail, THE System SHALL log the final error and continue operation (not block other processes)
4. WHERE an email fails (including authentication errors), THEN THE System SHALL log the failure immediately but still attempt retries

### Requirement 6: Rate Limiting for Email Sending

**User Story:** As a user, I want to avoid SMTP throttling, so that my notifications are delivered reliably without being rejected.

#### Acceptance Criteria

1. WHERE email sending is enabled, THE System SHALL implement rate limiting based on the SMTP provider's guidelines
2. FOR Gmail, THE Email Notifier SHALL limit to 500 emails per 24 hours
3. FOR Outlook, THE Email Notifier SHALL limit to 30 emails per minute
4. FOR Zoho, THE Email Notifier SHALL limit to 100 emails per day for free accounts
5. THE Email Notifier SHALL track the number of emails sent and remaining quota, logging warnings when approaching limits

### Requirement 7: Email Template Customization

**User Story:** As an advanced user, I want to customize email templates, so that I can match the notifications to my preferences or branding.

#### Acceptance Criteria

1. WHERE a template directory is configured, THE System SHALL look for email template files in that directory
2. WHEN an HTML template file exists, THE System SHALL use it for HTML email generation
3. WHEN a plain text template file exists, THE System SHALL use it for plain text email generation
4. THE Template File Format SHALL support Go template syntax for dynamic content insertion
5. FOR ALL template fields, THE System SHALL require explicit values (no empty fields in rendered output)

### Requirement 8: Security and Authentication

**User Story:** As a security-conscious user, I want my email credentials to be handled securely, so that my account information is protected.

#### Acceptance Criteria

1. WHEN SMTP credentials are loaded, THE System SHALL never log the password field in plain text
2. WHERE credentials are required, THE System SHALL validate that both username and password are non-empty before attempting to send
3. THE Email Notifier SHALL use STARTTLS for all connections on port 587 and implicit TLS for port 465
4. FOR Gmail accounts, THE System SHALL require App Password authentication, not regular passwords
5. WHERE environment variables are used for credentials, THE System SHALL prefer them over config file values

### Requirement 9: Testing and Validation

**User Story:** As a developer, I want to test the email notification system, so that I can verify functionality without sending real emails.

#### Acceptance Criteria

1. WHEN email notifications are tested, THE System SHALL provide a mock SMTP server for unit testing
2. THE Unit Tests SHALL cover successful email sends, retry scenarios, and error handling
3. WHERE integration tests are run, THE System SHALL use test email accounts with known limits
4. FOR retry logic tests, THE System SHALL simulate transient failures (timeouts, rate limits)
5. THE Test Suite SHALL include a validation test that confirms all required config fields are present

### Requirement 10: Graceful Degradation

**User Story:** As a user, I want the application to continue working even if email notifications fail, so that I don't lose functionality.

#### Acceptance Criteria

1. WHEN email sending fails permanently, THE System SHALL log the error but continue checking for new chapters
2. WHERE email is disabled in configuration, THEN THE System SHALL skip all notification attempts and not attempt to record failures
3. THE System SHALL record failed notifications in the database for later review when email is enabled
4. FOR CLI commands that trigger notifications, THE System SHALL report success/failure status without exiting

## Common Acceptance Criteria Patterns Applied

### Invariants
- Email notifier always starts in a valid state (validated config)
- Template rendering never panics (graceful fallback to defaults)

### Round Trip Properties
- Parse and re-serialize email config round-trip correctly
- Email template + data → HTML → parse HTML → verify data

### Idempotence
- Calling SendNewChapter multiple times with same data sends same notification
- Rate limiter state resets on application restart (expected behavior)

### Metamorphic Properties
- After retry exhaustion, error message contains original error details
- Total notification count (success + failed + skipped) equals total processed

### Error Conditions
- Missing SMTP host returns clear validation error
- Invalid template path falls back to built-in templates
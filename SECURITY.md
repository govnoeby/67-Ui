# Security Policy

## Supported Versions

| Version | Supported |
|---|---|
| latest | ✅ |

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability in 67-Ui, please do **not** open a public issue.

Instead, report it privately via one of these methods:

- **GitHub Security Advisories**: Use the [Report a Vulnerability](https://github.com/govnoeby/67-Ui/security/advisories/new) feature
- **Email**: Open a GitHub issue requesting contact information

We will acknowledge receipt within 48 hours and provide a timeline for a fix.

## Scope

The following are considered in scope:
- Authentication bypass
- Remote code execution
- SQL injection
- Cross-site scripting (XSS)
- Cross-site request forgery (CSRF)
- Sensitive data exposure
- Authentication or authorization flaws

## Best Practices

- Always change the default admin password
- Enable HTTPS with valid SSL certificates
- Enable two-factor authentication (2FA)
- Keep the panel updated to the latest version
- Use fail2ban for brute-force protection
- Restrict panel access by domain where possible

See the [Security](https://github.com/govnoeby/67-Ui/wiki/Security) wiki page for more details.

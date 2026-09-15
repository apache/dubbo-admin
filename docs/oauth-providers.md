# OAuth and OIDC Providers

Dubbo Admin supports password authentication, GitHub OAuth, and providers that
implement standard OpenID Connect (OIDC). OAuth/OIDC providers are configured
under `console.auth.providers`.

## Provider Support

| Provider | Configuration type | Status | Notes |
| --- | --- | --- | --- |
| GitHub | `github` | Supported | Uses GitHub's OAuth endpoints and user APIs. |
| Google | `oidc` | Supported | Uses standard OIDC discovery, ID token validation, and UserInfo. |
| Keycloak, Okta, Azure AD, and other compliant providers | `oidc` | Supported in principle | Works when the provider implements standard OIDC discovery and the configured claims are available. |
| Non-OIDC OAuth providers | Not available | Not supported generically | A provider-specific adapter is required. |

GitHub has a dedicated adapter because GitHub OAuth does not provide the OIDC
discovery document or a standard ID token. Other identity platforms should use
`type: oidc` when they implement standard OIDC.

## Example Configuration

See [`app/dubbo-admin/dubbo-admin-oauth-example.yaml`](../app/dubbo-admin/dubbo-admin-oauth-example.yaml)
for a complete, sanitized example. Replace all `replace-with-*` values before
using it. Do not commit client secrets or other production credentials.

For each enabled provider, register the exact `redirectUrl` with the identity
provider. The callback path is:

```text
/api/v1/auth/providers/{provider-id}/callback
```

Use HTTPS and set `sessionCookieSecure: true` in production. OIDC discovery
endpoints must also use HTTPS in release mode.

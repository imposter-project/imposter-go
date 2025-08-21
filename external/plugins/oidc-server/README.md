# OIDC Server Plugin

An OpenID Connect authorization server implementation as an external plugin for Imposter. This plugin provides a complete OIDC authorization server with support for the Authorization Code flow, PKCE, and standard OIDC endpoints.

## Features

- **OpenID Connect Authorization Code Flow** with full RFC compliance
- **PKCE Support** (RFC 7636) with both S256 and plain code challenge methods
- **Standard OIDC Endpoints**:
  - `/.well-known/openid-configuration` - OIDC Discovery
  - `/oidc/authorize` - Authorization endpoint
  - `/oidc/token` - Token endpoint
  - `/oidc/userinfo` - Userinfo endpoint
- **Web-based User Authentication** with responsive HTML login form
- **JWT Token Generation** with HS256 signing
- **Configurable Users and Clients** via YAML configuration files
- **Standard OIDC Scopes**: `openid`, `profile`, `email`, `address`, `phone`
- **Client Authentication** with client secrets
- **State and Nonce Parameter Support**

## File Structure

```
external/plugins/oidc-server/
├── main.go                 # Plugin entry point and handshake
├── plugin.go              # Core plugin implementation and routing
├── auth.go                # Authorization endpoint logic
├── token.go               # Token endpoint and JWT handling
├── userinfo.go            # Userinfo endpoint
├── config.go              # Configuration loading and validation
├── users.go               # User authentication and claims
├── pkce.go                # PKCE implementation
├── templates/             # HTML templates
│   ├── login.html         # Login form template
│   └── error.html         # Error page template
└── README.md              # This file
```

## Configuration

The plugin automatically uses the server URL configured in Imposter for OIDC discovery metadata and JWT issuer claims. It loads user and client configuration from `oidc-users.yaml` (or `oidc-config.yaml`) in the configuration directory:

```yaml
users:
  - username: "alice"
    password: "password123"
    claims:
      sub: "alice"
      email: "alice@example.com"
      given_name: "Alice"
      family_name: "Smith"
      name: "Alice Smith"
      preferred_username: "alice"
  - username: "bob"
    password: "secret456"
    claims:
      sub: "bob"
      email: "bob@company.com"
      given_name: "Bob"
      family_name: "Jones"
      name: "Bob Jones"

clients:
  - client_id: "webapp"
    client_secret: "webapp-secret"
    redirect_uris:
      - "http://localhost:3000/callback"
      - "http://localhost:8080/callback"
  - client_id: "mobile-app"
    redirect_uris:
      - "com.example.app://oauth/callback"
      - "http://localhost:8080/mobile-callback"
```

## Usage

### 1. Enable External Plugins

```bash
export IMPOSTER_EXTERNAL_PLUGINS=true
```

### 2. Create Configuration

Create an `imposter-config.yaml`:

```yaml
plugin: oidc-server
resources: []
```

And an `oidc-users.yaml` with your users and clients (see example above).

### 3. Run Imposter

```bash
make run /path/to/your/config
```

## API Endpoints

### OIDC Discovery

**Endpoint:** `GET /.well-known/openid-configuration`

```bash
curl http://localhost:8080/.well-known/openid-configuration
```

**Response:**
```json
{
  "issuer": "http://localhost:8080",
  "authorization_endpoint": "http://localhost:8080/oidc/authorize",
  "token_endpoint": "http://localhost:8080/oidc/token",
  "userinfo_endpoint": "http://localhost:8080/oidc/userinfo",
  "response_types_supported": ["code"],
  "subject_types_supported": ["public"],
  "id_token_signing_alg_values_supported": ["HS256"],
  "scopes_supported": ["openid", "profile", "email"],
  "claims_supported": ["sub", "name", "given_name", "family_name", "email"],
  "code_challenge_methods_supported": ["S256", "plain"]
}
```

**Note:** The URLs in the discovery document will automatically use the server URL configured in Imposter, not hardcoded localhost addresses.

### Authorization Endpoint

**Endpoint:** `GET /oidc/authorize`

**Parameters:**
- `client_id` (required) - Client identifier
- `redirect_uri` (required) - Redirect URI after authorization
- `response_type` (required) - Must be "code"
- `scope` (required) - Must include "openid", can include "profile", "email"
- `state` (optional) - State parameter for CSRF protection
- `nonce` (optional) - Nonce for ID token validation
- `code_challenge` (optional) - PKCE code challenge
- `code_challenge_method` (optional) - "S256" or "plain"

**Example Request:**
```bash
# Basic Authorization Request
curl "http://localhost:8080/oidc/authorize?client_id=webapp&redirect_uri=http://localhost:8080/callback&response_type=code&scope=openid+profile+email&state=xyz123"
```

**Example with PKCE:**
```bash
# First generate code verifier and challenge (example using Python)
python3 -c "
import base64, hashlib, secrets
verifier = base64.urlsafe_b64encode(secrets.token_bytes(32)).decode().rstrip('=')
challenge = base64.urlsafe_b64encode(hashlib.sha256(verifier.encode()).digest()).decode().rstrip('=')
print(f'Verifier: {verifier}')
print(f'Challenge: {challenge}')
"

# Use the challenge in authorization request
curl "http://localhost:8080/oidc/authorize?client_id=webapp&redirect_uri=http://localhost:8080/callback&response_type=code&scope=openid+profile&code_challenge=<challenge>&code_challenge_method=S256"
```

This will redirect to the login form. After successful login, you'll be redirected to:
```
http://localhost:8080/callback?code=<authorization_code>&state=xyz123
```

### Token Endpoint

**Endpoint:** `POST /oidc/token`

**Parameters (form-encoded):**
- `grant_type` (required) - Must be "authorization_code"
- `client_id` (required) - Client identifier
- `client_secret` (optional) - Client secret if configured
- `code` (required) - Authorization code from authorize endpoint
- `redirect_uri` (required) - Same redirect URI used in authorization
- `code_verifier` (optional) - PKCE code verifier if PKCE was used

**Example Request:**
```bash
curl -X POST http://localhost:8080/oidc/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code&client_id=webapp&client_secret=webapp-secret&code=<auth_code>&redirect_uri=http://localhost:8080/callback"
```

**Example with PKCE:**
```bash
curl -X POST http://localhost:8080/oidc/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code&client_id=webapp&code=<auth_code>&redirect_uri=http://localhost:8080/callback&code_verifier=<code_verifier>"
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "id_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "openid profile email"
}
```

### Userinfo Endpoint

**Endpoint:** `GET /oidc/userinfo`

**Headers:**
- `Authorization: Bearer <access_token>`

**Example Request:**
```bash
curl -H "Authorization: Bearer <access_token>" \
  http://localhost:8080/oidc/userinfo
```

**Response:**
```json
{
  "sub": "alice",
  "name": "Alice Smith",
  "given_name": "Alice",
  "family_name": "Smith",
  "email": "alice@example.com"
}
```

## Complete Authorization Code Flow Example

### 1. Start Authorization Flow
```bash
# Navigate to this URL in a browser
open "http://localhost:8080/oidc/authorize?client_id=webapp&redirect_uri=http://localhost:8080/callback&response_type=code&scope=openid+profile+email&state=test123"
```

### 2. Login
- Enter username: `alice`
- Enter password: `password123`
- Click "Sign In"

### 3. Extract Authorization Code
After login, you'll be redirected to:
```
http://localhost:8080/callback?code=<authorization_code>&state=test123
```

### 4. Exchange Code for Tokens
```bash
curl -X POST http://localhost:8080/oidc/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code&client_id=webapp&client_secret=webapp-secret&code=<authorization_code>&redirect_uri=http://localhost:8080/callback"
```

### 5. Use Access Token
```bash
curl -H "Authorization: Bearer <access_token>" \
  http://localhost:8080/oidc/userinfo
```

## PKCE Flow Example

### 1. Generate PKCE Parameters
```javascript
// JavaScript (can be run in browser console)
function generatePKCE() {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  const verifier = btoa(String.fromCharCode.apply(null, array))
    .replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
  
  const encoder = new TextEncoder();
  const data = encoder.encode(verifier);
  return crypto.subtle.digest('SHA-256', data).then(hash => {
    const challenge = btoa(String.fromCharCode.apply(null, new Uint8Array(hash)))
      .replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
    return { verifier, challenge };
  });
}

generatePKCE().then(({verifier, challenge}) => {
  console.log('Code Verifier:', verifier);
  console.log('Code Challenge:', challenge);
});
```

### 2. Authorization Request with PKCE
```bash
# Use the generated challenge
open "http://localhost:8080/oidc/authorize?client_id=mobile-app&redirect_uri=http://localhost:8080/mobile-callback&response_type=code&scope=openid+profile&code_challenge=<challenge>&code_challenge_method=S256&state=mobile123"
```

### 3. Token Request with PKCE
```bash
curl -X POST http://localhost:8080/oidc/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code&client_id=mobile-app&code=<authorization_code>&redirect_uri=http://localhost:8080/mobile-callback&code_verifier=<verifier>"
```

## Testing with JWT.io

1. Start the authorization flow with `redirect_uri=https://jwt.io`
2. Complete login
3. Copy the authorization code from the JWT.io URL
4. Exchange for tokens
5. Copy the `id_token` and `access_token` to JWT.io to inspect claims

## Error Handling

The plugin provides detailed error responses following OIDC specifications:

### Authorization Errors
- `invalid_request` - Missing or invalid parameters
- `invalid_client` - Unknown client ID
- `invalid_scope` - Invalid scope parameter
- `unsupported_response_type` - Only "code" is supported

### Token Errors
- `invalid_grant` - Invalid or expired authorization code
- `invalid_client` - Invalid client credentials
- `unsupported_grant_type` - Only "authorization_code" is supported

### Userinfo Errors
- `invalid_token` - Invalid or expired access token

## Security Considerations

- **JWT Signing**: Tokens are signed with HS256 using a randomly generated secret
- **Code Expiration**: Authorization codes expire after 10 minutes
- **Token Expiration**: Access tokens expire after 1 hour
- **Session Management**: Login sessions expire after 10 minutes
- **PKCE Support**: Recommended for public clients and enhanced security
- **Password Storage**: Example uses plain text - use bcrypt in production

## Default Configuration

If no configuration file is found, the plugin uses these defaults:

**Users:**
- alice / password
- bob / password

**Client:**
- client_id: test-client
- client_secret: test-secret
- redirect_uris: http://localhost:8080/callback, http://localhost:3000/callback

## Implementation Notes

- **Query Parameter Handling**: The plugin uses the `args.Query` field from `HandlerRequest` to access query parameters, providing clean access to parsed URL query values
- **Server URL Integration**: Automatically uses the server URL from Imposter's configuration for OIDC discovery metadata and JWT issuer claims
- **Thread Safety**: Uses mutex locks for concurrent access to sessions, authorization codes, and access tokens

## Building

The plugin is built as part of the main Imposter build:

```bash
make build-plugins
```

This creates `bin/plugin-oidc-server` which is automatically discovered by Imposter when external plugins are enabled.
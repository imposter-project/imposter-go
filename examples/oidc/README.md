# OIDC Server Plugin Example

This example demonstrates the `oidc-server` external plugin that provides OpenID Connect authorization server functionality.

## Features

- **Authorization Code Flow** with optional PKCE support
- **Standard OIDC Endpoints**:
  - `/.well-known/openid-configuration` - OIDC discovery
  - `/oidc/authorize` - Authorization endpoint
  - `/oidc/token` - Token endpoint  
  - `/oidc/userinfo` - Userinfo endpoint
- **Web-based Authentication** with username/password form
- **Configurable Users and Clients** via YAML configuration
- **JWT Token Support** with HS256 signing
- **Standard OIDC Scopes**: `openid`, `profile`, `email`

## Configuration

The OIDC server plugin is configured using the `config` block within your main Imposter configuration file:

### Complete Configuration (`imposter-config.yaml`)

```yaml
plugin: oidc-server
resources: []

config:
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
      password: "password456"
      claims:
        sub: "bob"
        email: "bob@example.com"
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

1. **Enable External Plugins**:
   ```bash
   export IMPOSTER_EXTERNAL_PLUGINS=true
   ```

2. **Run Imposter**:
   ```bash
   make run ./examples/oidc
   ```

3. **Test Authorization Flow**:
   Navigate to:
   ```
   http://localhost:8080/oidc/authorize?client_id=webapp&redirect_uri=http://localhost:8080/callback&response_type=code&scope=openid+profile+email&state=test123
   ```

4. **Login Credentials**:
   - Username: `alice` / Password: `password123`
   - Username: `bob` / Password: `password456`

## OIDC Flow Example

### 1. Authorization Request
```
GET /oidc/authorize?client_id=webapp&redirect_uri=http://localhost:8080/callback&response_type=code&scope=openid+profile+email&state=test123
```

### 2. User Login
Users will see a web form to enter username/password.

### 3. Authorization Response
```
HTTP/1.1 302 Found
Location: http://localhost:8080/callback?code=abc123&state=test123
```

### 4. Token Request
```bash
curl -X POST http://localhost:8080/oidc/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code&client_id=webapp&client_secret=webapp-secret&code=abc123&redirect_uri=http://localhost:8080/callback"
```

### 5. Token Response
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "id_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "openid profile email"
}
```

### 6. Userinfo Request
```bash
curl -H "Authorization: Bearer <access_token>" http://localhost:8080/oidc/userinfo
```

### 7. Userinfo Response
```json
{
  "sub": "alice",
  "name": "Alice Smith",
  "given_name": "Alice",
  "family_name": "Smith",
  "email": "alice@example.com"
}
```

## PKCE Support

The plugin supports PKCE (RFC 7636) for enhanced security:

1. **Generate Code Verifier and Challenge**:
   ```javascript
   // JavaScript example
   const codeVerifier = base64URLEncode(crypto.getRandomValues(new Uint8Array(32)));
   const challenge = base64URLEncode(sha256(codeVerifier));
   ```

2. **Authorization Request with PKCE**:
   ```
   GET /oidc/authorize?client_id=webapp&redirect_uri=http://localhost:8080/callback&response_type=code&scope=openid&code_challenge=<challenge>&code_challenge_method=S256
   ```

3. **Token Request with PKCE**:
   ```bash
   curl -X POST http://localhost:8080/oidc/token \
     -d "grant_type=authorization_code&client_id=webapp&code=abc123&redirect_uri=http://localhost:8080/callback&code_verifier=<verifier>"
   ```

## Discovery Document

The OIDC discovery document is available at:
```
GET /.well-known/openid-configuration
```

This provides metadata about the authorization server endpoints and capabilities.

## Security Notes

- JWT tokens are signed with HS256 using a randomly generated secret
- Passwords in the example use plain text for simplicity - use bcrypt hashed passwords in production
- Authorization codes and access tokens have configurable expiration times
- PKCE is supported for enhanced security with public clients
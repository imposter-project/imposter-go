package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	os.Exit(code)
}

func resetResourceCounter(t *testing.T) {
	t.Helper()
	resourceCounter = 0
}

func TestTransformSecurityConfig_NoSecurity(t *testing.T) {
	resetResourceCounter(t)
	cfg := &Config{
		Plugin: "rest",
		Resources: []Resource{
			{Response: Response{StatusCode: 200}},
		},
	}

	transformSecurityConfig(cfg)
	require.Empty(t, cfg.Interceptors)
}

func TestTransformSecurityConfig_SingleCondition(t *testing.T) {
	resetResourceCounter(t)
	cfg := &Config{
		Plugin: "rest",
		Security: &SecurityConfig{
			Default: "Deny",
			Conditions: []SecurityCondition{
				{
					Effect: "Permit",
					RequestHeaders: map[string]MatcherUnmarshaler{
						"Authorization": {
							Matcher: StringMatcher("Bearer token"),
						},
					},
				},
			},
		},
	}

	transformSecurityConfig(cfg)

	// Check that security config was removed
	require.Nil(t, cfg.Security)

	// Should have 2 interceptors: one for the condition and one for default deny
	require.Len(t, cfg.Interceptors, 2)

	// Check condition interceptor
	interceptor := cfg.Interceptors[0]
	require.Contains(t, interceptor.Headers, "Authorization")
	matcher := interceptor.Headers["Authorization"].Matcher
	require.IsType(t, StringMatcher(""), matcher)
	require.Equal(t, StringMatcher("Bearer token"), matcher)
	require.Contains(t, interceptor.Capture, "security_condition1")
	require.Equal(t, "request", interceptor.Capture["security_condition1"].Store)
	require.Equal(t, "met", interceptor.Capture["security_condition1"].Const)
	require.True(t, interceptor.Continue)

	// Check deny interceptor
	deny := cfg.Interceptors[1]
	require.Len(t, deny.AnyOf, 1)
	require.Equal(t, "${stores.request.security_condition1}", deny.AnyOf[0].Expression)
	require.Equal(t, "met", deny.AnyOf[0].MatchCondition.Value)
	require.Equal(t, "NotEqualTo", deny.AnyOf[0].MatchCondition.Operator)
	require.Equal(t, 401, deny.Response.StatusCode)
	require.Equal(t, "Unauthorised", deny.Response.Content)
	require.Equal(t, "text/plain", deny.Response.Headers["Content-Type"])
	require.False(t, deny.Continue)
}

func TestTransformSecurityConfig_AllConditionTypes(t *testing.T) {
	resetResourceCounter(t)
	cfg := &Config{
		Plugin: "rest",
		Security: &SecurityConfig{
			Default: "Deny",
			Conditions: []SecurityCondition{
				{
					Effect: "Permit",
					RequestHeaders: map[string]MatcherUnmarshaler{
						"Authorization": {
							Matcher: StringMatcher("Bearer token"),
						},
					},
					QueryParams: map[string]MatcherUnmarshaler{
						"apiKey": {
							Matcher: StringMatcher("secret"),
						},
					},
					FormParams: map[string]MatcherUnmarshaler{
						"token": {
							Matcher: StringMatcher("form-token"),
						},
					},
				},
			},
		},
	}

	transformSecurityConfig(cfg)

	// Check condition interceptor
	interceptor := cfg.Interceptors[0]

	// Check headers
	require.Contains(t, interceptor.Headers, "Authorization")
	authMatcher := interceptor.Headers["Authorization"].Matcher
	require.IsType(t, StringMatcher(""), authMatcher)
	require.Equal(t, StringMatcher("Bearer token"), authMatcher)

	// Check query params
	require.Contains(t, interceptor.QueryParams, "apiKey")
	apiKeyMatcher := interceptor.QueryParams["apiKey"].Matcher
	require.IsType(t, StringMatcher(""), apiKeyMatcher)
	require.Equal(t, StringMatcher("secret"), apiKeyMatcher)

	// Check form params
	require.Contains(t, interceptor.FormParams, "token")
	tokenMatcher := interceptor.FormParams["token"].Matcher
	require.IsType(t, StringMatcher(""), tokenMatcher)
	require.Equal(t, StringMatcher("form-token"), tokenMatcher)
}

func TestTransformSecurityConfig_DefaultPermit(t *testing.T) {
	resetResourceCounter(t)
	cfg := &Config{
		Plugin: "rest",
		Security: &SecurityConfig{
			Default: "Permit",
			Conditions: []SecurityCondition{
				{
					Effect: "Deny",
					RequestHeaders: map[string]MatcherUnmarshaler{
						"Authorization": {
							Matcher: StringMatcher("Bearer token"),
						},
					},
				},
			},
		},
	}

	transformSecurityConfig(cfg)

	// Should only have the condition interceptor, no default deny
	require.Len(t, cfg.Interceptors, 1)

	// Check condition interceptor
	interceptor := cfg.Interceptors[0]
	require.Contains(t, interceptor.Headers, "Authorization")
	authMatcher := interceptor.Headers["Authorization"].Matcher
	require.IsType(t, StringMatcher(""), authMatcher)
	require.Equal(t, StringMatcher("Bearer token"), authMatcher)
}

func TestBuildSecurityEvalConditions(t *testing.T) {
	conditions := buildSecurityEvalConditions(0, "")
	require.Empty(t, conditions)

	conditions = buildSecurityEvalConditions(1, "")
	require.Len(t, conditions, 1)
	require.Equal(t, "${stores.request.security_condition1}", conditions[0].Expression)
	require.Equal(t, "met", conditions[0].MatchCondition.Value)
	require.Equal(t, "NotEqualTo", conditions[0].MatchCondition.Operator)

	conditions = buildSecurityEvalConditions(2, "prefix_")
	require.Len(t, conditions, 2)
	require.Equal(t, "${stores.request.prefix_security_condition1}", conditions[0].Expression)
	require.Equal(t, "met", conditions[0].MatchCondition.Value)
	require.Equal(t, "NotEqualTo", conditions[0].MatchCondition.Operator)
	require.Equal(t, "${stores.request.prefix_security_condition2}", conditions[1].Expression)
	require.Equal(t, "met", conditions[1].MatchCondition.Value)
	require.Equal(t, "NotEqualTo", conditions[1].MatchCondition.Operator)
}

// Keep the existing integration test
func TestLoadConfig_WithSecurity(t *testing.T) {
	resetResourceCounter(t)
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a test config file with security configuration
	configContent := `plugin: rest
security:
  default: Deny
  conditions:
  - effect: Permit
    requestHeaders:
      Authorization: s3cr3t
    queryParams:
      apiKey: key123
    formParams:
      token: token456
  - effect: Permit
    requestHeaders:
      X-API-Key: key789
    queryParams:
      version: v1
    formParams:
      client: web
resources:
  - path: /test
    response:
      content: test response
      statusCode: 200`

	err := os.WriteFile(filepath.Join(tempDir, "test-config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	// Load the config
	configs := LoadConfig(tempDir)
	require.Len(t, configs, 1)

	cfg := configs[0]
	require.Equal(t, "rest", cfg.Plugin)
	require.Nil(t, cfg.Security, "Security config should be transformed and removed")
	require.Len(t, cfg.Resources, 1)
	require.Len(t, cfg.Interceptors, 3) // 2 conditions + 1 default deny

	// Check first condition interceptor
	authInterceptor := cfg.Interceptors[0]
	// Check headers
	require.Contains(t, authInterceptor.Headers, "Authorization")
	authMatcher := authInterceptor.Headers["Authorization"].Matcher
	require.IsType(t, StringMatcher(""), authMatcher)
	require.Equal(t, StringMatcher("s3cr3t"), authMatcher)
	// Check query params
	require.Contains(t, authInterceptor.QueryParams, "apiKey")
	apiKeyMatcher := authInterceptor.QueryParams["apiKey"].Matcher
	require.IsType(t, StringMatcher(""), apiKeyMatcher)
	require.Equal(t, StringMatcher("key123"), apiKeyMatcher)
	// Check form params
	require.Contains(t, authInterceptor.FormParams, "token")
	tokenMatcher := authInterceptor.FormParams["token"].Matcher
	require.IsType(t, StringMatcher(""), tokenMatcher)
	require.Equal(t, StringMatcher("token456"), tokenMatcher)
	// Check capture
	require.Contains(t, authInterceptor.Capture, "security_condition1")
	require.Equal(t, "request", authInterceptor.Capture["security_condition1"].Store)
	require.Equal(t, "met", authInterceptor.Capture["security_condition1"].Const)
	require.True(t, authInterceptor.Continue)

	// Check second condition interceptor
	apiKeyInterceptor := cfg.Interceptors[1]
	// Check headers
	require.Contains(t, apiKeyInterceptor.Headers, "X-API-Key")
	xapiKeyMatcher := apiKeyInterceptor.Headers["X-API-Key"].Matcher
	require.IsType(t, StringMatcher(""), xapiKeyMatcher)
	require.Equal(t, StringMatcher("key789"), xapiKeyMatcher)
	// Check query params
	require.Contains(t, apiKeyInterceptor.QueryParams, "version")
	versionMatcher := apiKeyInterceptor.QueryParams["version"].Matcher
	require.IsType(t, StringMatcher(""), versionMatcher)
	require.Equal(t, StringMatcher("v1"), versionMatcher)
	// Check form params
	require.Contains(t, apiKeyInterceptor.FormParams, "client")
	clientMatcher := apiKeyInterceptor.FormParams["client"].Matcher
	require.IsType(t, StringMatcher(""), clientMatcher)
	require.Equal(t, StringMatcher("web"), clientMatcher)
	// Check capture
	require.Contains(t, apiKeyInterceptor.Capture, "security_condition2")
	require.Equal(t, "request", apiKeyInterceptor.Capture["security_condition2"].Store)
	require.Equal(t, "met", apiKeyInterceptor.Capture["security_condition2"].Const)
	require.True(t, apiKeyInterceptor.Continue)

	// Check deny interceptor
	denyInterceptor := cfg.Interceptors[2]
	require.Len(t, denyInterceptor.AnyOf, 2)
	// Check first eval condition
	require.Equal(t, "${stores.request.security_condition1}", denyInterceptor.AnyOf[0].Expression)
	require.Equal(t, "met", denyInterceptor.AnyOf[0].MatchCondition.Value)
	require.Equal(t, "NotEqualTo", denyInterceptor.AnyOf[0].MatchCondition.Operator)

	// Check second eval condition
	require.Equal(t, "${stores.request.security_condition2}", denyInterceptor.AnyOf[1].Expression)
	require.Equal(t, "met", denyInterceptor.AnyOf[1].MatchCondition.Value)
	require.Equal(t, "NotEqualTo", denyInterceptor.AnyOf[1].MatchCondition.Operator)

	require.NotNil(t, denyInterceptor.Response)
	require.Equal(t, 401, denyInterceptor.Response.StatusCode)
	require.Equal(t, "Unauthorised", denyInterceptor.Response.Content)
	require.Equal(t, "text/plain", denyInterceptor.Response.Headers["Content-Type"])
	require.False(t, denyInterceptor.Continue)
}

func TestTransformSecurityConfig_AllOperators(t *testing.T) {
	resetResourceCounter(t)
	cfg := &Config{
		Plugin: "rest",
		Security: &SecurityConfig{
			Default: "Deny",
			Conditions: []SecurityCondition{
				{
					Effect: "Permit",
					RequestHeaders: map[string]MatcherUnmarshaler{
						"Authorization": {
							Matcher: MatchCondition{
								Value:    "Bearer .*",
								Operator: "Matches",
							},
						},
						"X-API-Key": {
							Matcher: MatchCondition{
								Value:    "secret",
								Operator: "NotEqualTo",
							},
						},
						"X-Custom": {
							Matcher: MatchCondition{
								Operator: "Exists",
							},
						},
						"X-Other": {
							Matcher: MatchCondition{
								Operator: "NotExists",
							},
						},
					},
					QueryParams: map[string]MatcherUnmarshaler{
						"version": {
							Matcher: MatchCondition{
								Value:    "v2",
								Operator: "Contains",
							},
						},
						"debug": {
							Matcher: MatchCondition{
								Value:    "true",
								Operator: "NotContains",
							},
						},
					},
					FormParams: map[string]MatcherUnmarshaler{
						"token": {
							Matcher: MatchCondition{
								Value:    "^token-\\d+$",
								Operator: "NotMatches",
							},
						},
					},
				},
			},
		},
	}

	transformSecurityConfig(cfg)

	// Check condition interceptor
	interceptor := cfg.Interceptors[0]

	// Check headers
	require.Contains(t, interceptor.Headers, "Authorization")
	auth := interceptor.Headers["Authorization"].Matcher.(MatchCondition)
	require.Equal(t, "Bearer .*", auth.Value)
	require.Equal(t, "Matches", auth.Operator)

	require.Contains(t, interceptor.Headers, "X-API-Key")
	apiKey := interceptor.Headers["X-API-Key"].Matcher.(MatchCondition)
	require.Equal(t, "secret", apiKey.Value)
	require.Equal(t, "NotEqualTo", apiKey.Operator)

	require.Contains(t, interceptor.Headers, "X-Custom")
	custom := interceptor.Headers["X-Custom"].Matcher.(MatchCondition)
	require.Equal(t, "", custom.Value)
	require.Equal(t, "Exists", custom.Operator)

	require.Contains(t, interceptor.Headers, "X-Other")
	other := interceptor.Headers["X-Other"].Matcher.(MatchCondition)
	require.Equal(t, "", other.Value)
	require.Equal(t, "NotExists", other.Operator)

	// Check query params
	require.Contains(t, interceptor.QueryParams, "version")
	version := interceptor.QueryParams["version"].Matcher.(MatchCondition)
	require.Equal(t, "v2", version.Value)
	require.Equal(t, "Contains", version.Operator)

	require.Contains(t, interceptor.QueryParams, "debug")
	debug := interceptor.QueryParams["debug"].Matcher.(MatchCondition)
	require.Equal(t, "true", debug.Value)
	require.Equal(t, "NotContains", debug.Operator)

	// Check form params
	require.Contains(t, interceptor.FormParams, "token")
	token := interceptor.FormParams["token"].Matcher.(MatchCondition)
	require.Equal(t, "^token-\\d+$", token.Value)
	require.Equal(t, "NotMatches", token.Operator)
}

func TestTransformSecurityConfig_ResourceLevel(t *testing.T) {
	resetResourceCounter(t)
	cfg := &Config{
		Plugin: "rest",
		Resources: []Resource{
			{
				RequestMatcher: RequestMatcher{
					Path: "/protected",
				},
				Response: Response{
					StatusCode: 200,
				},
				Security: &SecurityConfig{
					Default: "Deny",
					Conditions: []SecurityCondition{
						{
							Effect: "Permit",
							RequestHeaders: map[string]MatcherUnmarshaler{
								"Authorization": {
									Matcher: StringMatcher("Bearer token"),
								},
							},
						},
					},
				},
			},
			{
				RequestMatcher: RequestMatcher{
					Path: "/also-protected",
				},
				Response: Response{
					StatusCode: 200,
				},
				Security: &SecurityConfig{
					Default: "Deny",
					Conditions: []SecurityCondition{
						{
							Effect: "Permit",
							QueryParams: map[string]MatcherUnmarshaler{
								"apiKey": {
									Matcher: StringMatcher("secret"),
								},
							},
						},
					},
				},
			},
		},
	}

	transformSecurityConfig(cfg)

	// Check that security configs were removed
	require.Nil(t, cfg.Resources[0].Security)
	require.Nil(t, cfg.Resources[1].Security)

	// Should have 4 interceptors: one condition + one deny for each resource
	require.Len(t, cfg.Interceptors, 4)

	// Check first resource's condition interceptor
	interceptor1 := cfg.Interceptors[0]
	require.Contains(t, interceptor1.Headers, "Authorization")
	authMatcher := interceptor1.Headers["Authorization"].Matcher
	require.IsType(t, StringMatcher(""), authMatcher)
	require.Equal(t, StringMatcher("Bearer token"), authMatcher)
	require.Contains(t, interceptor1.Capture, "resource1_security_condition1")
	require.Equal(t, "request", interceptor1.Capture["resource1_security_condition1"].Store)
	require.Equal(t, "met", interceptor1.Capture["resource1_security_condition1"].Const)
	require.True(t, interceptor1.Continue)

	// Check first resource's deny interceptor
	deny1 := cfg.Interceptors[1]
	require.Len(t, deny1.AnyOf, 1)
	require.Equal(t, "${stores.request.resource1_security_condition1}", deny1.AnyOf[0].Expression)
	require.Equal(t, "met", deny1.AnyOf[0].MatchCondition.Value)
	require.Equal(t, "NotEqualTo", deny1.AnyOf[0].MatchCondition.Operator)
	require.Equal(t, 401, deny1.Response.StatusCode)
	require.Equal(t, "Unauthorised", deny1.Response.Content)
	require.Equal(t, "text/plain", deny1.Response.Headers["Content-Type"])
	require.False(t, deny1.Continue)

	// Check second resource's condition interceptor
	interceptor2 := cfg.Interceptors[2]
	require.Contains(t, interceptor2.QueryParams, "apiKey")
	apiKeyMatcher := interceptor2.QueryParams["apiKey"].Matcher
	require.IsType(t, StringMatcher(""), apiKeyMatcher)
	require.Equal(t, StringMatcher("secret"), apiKeyMatcher)
	require.Contains(t, interceptor2.Capture, "resource2_security_condition1")
	require.Equal(t, "request", interceptor2.Capture["resource2_security_condition1"].Store)
	require.Equal(t, "met", interceptor2.Capture["resource2_security_condition1"].Const)
	require.True(t, interceptor2.Continue)

	// Check second resource's deny interceptor
	deny2 := cfg.Interceptors[3]
	require.Len(t, deny2.AnyOf, 1)
	require.Equal(t, "${stores.request.resource2_security_condition1}", deny2.AnyOf[0].Expression)
	require.Equal(t, "met", deny2.AnyOf[0].MatchCondition.Value)
	require.Equal(t, "NotEqualTo", deny2.AnyOf[0].MatchCondition.Operator)
	require.Equal(t, 401, deny2.Response.StatusCode)
	require.Equal(t, "Unauthorised", deny2.Response.Content)
	require.Equal(t, "text/plain", deny2.Response.Headers["Content-Type"])
	require.False(t, deny2.Continue)
}

func TestTransformSecurityConfig_BothLevels(t *testing.T) {
	resetResourceCounter(t)
	cfg := &Config{
		Plugin: "rest",
		Security: &SecurityConfig{
			Default: "Deny",
			Conditions: []SecurityCondition{
				{
					Effect: "Permit",
					RequestHeaders: map[string]MatcherUnmarshaler{
						"X-API-Key": {
							Matcher: StringMatcher("global-key"),
						},
					},
				},
			},
		},
		Resources: []Resource{
			{
				RequestMatcher: RequestMatcher{
					Path: "/extra-protected",
				},
				Response: Response{
					StatusCode: 200,
				},
				Security: &SecurityConfig{
					Default: "Deny",
					Conditions: []SecurityCondition{
						{
							Effect: "Permit",
							RequestHeaders: map[string]MatcherUnmarshaler{
								"Authorization": {
									Matcher: StringMatcher("Bearer token"),
								},
							},
						},
					},
				},
			},
		},
	}

	transformSecurityConfig(cfg)

	// Check that security configs were removed
	require.Nil(t, cfg.Security)
	require.Nil(t, cfg.Resources[0].Security)

	// Should have 4 interceptors:
	// - root level: condition + deny
	// - resource level: condition + deny
	require.Len(t, cfg.Interceptors, 4)

	// Check root level condition interceptor
	interceptor1 := cfg.Interceptors[0]
	require.Contains(t, interceptor1.Headers, "X-API-Key")
	apiKeyMatcher := interceptor1.Headers["X-API-Key"].Matcher
	require.IsType(t, StringMatcher(""), apiKeyMatcher)
	require.Equal(t, StringMatcher("global-key"), apiKeyMatcher)
	require.Contains(t, interceptor1.Capture, "security_condition1")
	require.Equal(t, "met", interceptor1.Capture["security_condition1"].Const)

	// Check root level deny interceptor
	deny1 := cfg.Interceptors[1]
	require.Equal(t, "${stores.request.security_condition1}", deny1.AnyOf[0].Expression)

	// Check resource level condition interceptor
	interceptor2 := cfg.Interceptors[2]
	require.Contains(t, interceptor2.Headers, "Authorization")
	authMatcher := interceptor2.Headers["Authorization"].Matcher
	require.IsType(t, StringMatcher(""), authMatcher)
	require.Equal(t, StringMatcher("Bearer token"), authMatcher)
	require.Contains(t, interceptor2.Capture, "resource1_security_condition1")
	require.Equal(t, "met", interceptor2.Capture["resource1_security_condition1"].Const)

	// Check resource level deny interceptor
	deny2 := cfg.Interceptors[3]
	require.Equal(t, "${stores.request.resource1_security_condition1}", deny2.AnyOf[0].Expression)
}

func TestTransformSecurityConfig_UniqueResourcePrefixes(t *testing.T) {
	resetResourceCounter(t)
	// Create first config with a protected resource
	cfg1 := &Config{
		Plugin: "rest",
		Resources: []Resource{
			{
				RequestMatcher: RequestMatcher{
					Path: "/protected1",
				},
				Security: &SecurityConfig{
					Default: "Deny",
					Conditions: []SecurityCondition{
						{
							Effect: "Permit",
							RequestHeaders: map[string]MatcherUnmarshaler{
								"Authorization": {
									Matcher: StringMatcher("token1"),
								},
							},
						},
					},
				},
			},
		},
	}

	// Create second config with another protected resource
	cfg2 := &Config{
		Plugin: "rest",
		Resources: []Resource{
			{
				RequestMatcher: RequestMatcher{
					Path: "/protected2",
				},
				Security: &SecurityConfig{
					Default: "Deny",
					Conditions: []SecurityCondition{
						{
							Effect: "Permit",
							RequestHeaders: map[string]MatcherUnmarshaler{
								"Authorization": {
									Matcher: StringMatcher("token2"),
								},
							},
						},
					},
				},
			},
		},
	}

	// Transform both configs
	transformSecurityConfig(cfg1)
	transformSecurityConfig(cfg2)

	// Check first config's interceptors
	require.Len(t, cfg1.Interceptors, 2) // condition + deny
	interceptor1 := cfg1.Interceptors[0]
	require.Contains(t, interceptor1.Capture, "resource1_security_condition1")
	deny1 := cfg1.Interceptors[1]
	require.Equal(t, "${stores.request.resource1_security_condition1}", deny1.AnyOf[0].Expression)

	// Check second config's interceptors
	require.Len(t, cfg2.Interceptors, 2) // condition + deny
	interceptor2 := cfg2.Interceptors[0]
	require.Contains(t, interceptor2.Capture, "resource2_security_condition1")
	deny2 := cfg2.Interceptors[1]
	require.Equal(t, "${stores.request.resource2_security_condition1}", deny2.AnyOf[0].Expression)

	// Create third config with multiple protected resources
	cfg3 := &Config{
		Plugin: "rest",
		Resources: []Resource{
			{
				RequestMatcher: RequestMatcher{
					Path: "/protected3a",
				},
				Security: &SecurityConfig{
					Default: "Deny",
					Conditions: []SecurityCondition{
						{
							Effect: "Permit",
							RequestHeaders: map[string]MatcherUnmarshaler{
								"Authorization": {
									Matcher: StringMatcher("token3a"),
								},
							},
						},
					},
				},
			},
			{
				RequestMatcher: RequestMatcher{
					Path: "/protected3b",
				},
				Security: &SecurityConfig{
					Default: "Deny",
					Conditions: []SecurityCondition{
						{
							Effect: "Permit",
							RequestHeaders: map[string]MatcherUnmarshaler{
								"Authorization": {
									Matcher: StringMatcher("token3b"),
								},
							},
						},
					},
				},
			},
		},
	}

	// Transform third config
	transformSecurityConfig(cfg3)

	// Check third config's interceptors
	require.Len(t, cfg3.Interceptors, 4) // 2 conditions + 2 denies
	interceptor3a := cfg3.Interceptors[0]
	require.Contains(t, interceptor3a.Capture, "resource3_security_condition1")
	deny3a := cfg3.Interceptors[1]
	require.Equal(t, "${stores.request.resource3_security_condition1}", deny3a.AnyOf[0].Expression)

	interceptor3b := cfg3.Interceptors[2]
	require.Contains(t, interceptor3b.Capture, "resource4_security_condition1")
	deny3b := cfg3.Interceptors[3]
	require.Equal(t, "${stores.request.resource4_security_condition1}", deny3b.AnyOf[0].Expression)
}

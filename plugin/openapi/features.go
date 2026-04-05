package openapi

import "github.com/imposter-project/imposter-go/pkg/feature"

// Feature flags controlling how the underlying libopenapi document is
// constructed. Both flags are read on first use of newOpenAPIParser.
//
// File references default to on: this matches the historical behaviour
// of the JVM engine, and resolving a local file imposes no new network
// surface.
//
// Remote references default to off: the JVM engine does allow them by
// default via setResolveFully(true), but outbound HTTP from a mock is
// undesirable in locked-down environments, so operators must opt in
// explicitly.
var (
	flagAllowFileRefs = feature.Register(feature.Flag{
		Name:        "openapi.allowFileRefs",
		EnvVar:      "IMPOSTER_OPENAPI_ALLOW_FILE_REFS",
		Default:     true,
		Description: "Allow OpenAPI $ref to resolve local file references.",
	})
	flagAllowRemoteRefs = feature.Register(feature.Flag{
		Name:        "openapi.allowRemoteRefs",
		EnvVar:      "IMPOSTER_OPENAPI_ALLOW_REMOTE_REFS",
		Default:     false,
		Description: "Allow OpenAPI $ref to resolve remote (http/https) references.",
	})
)

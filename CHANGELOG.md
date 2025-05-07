# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.28.0] - 2025-05-07
### Added
- feat: matched resources and interceptors can log a (templated) string.

### Changed
- test: use exchange object to minimise request args.

### Fixed
- fix: exit with nonzero status if no config files are found.

## [0.27.2] - 2025-04-18
### Changed
- refactor: introduce BaseResource to unify resource structure.

## [0.27.1] - 2025-04-15
### Changed
- chore(deps): bumps github.com/outofcoffee/go-xml-example-generator to v0.4.2.

## [0.27.0] - 2025-04-10
### Added
- feat(openapi): implement spec file resolution and remote retrieval.

## [0.26.0] - 2025-04-10
### Added
- feat: allows validation failure behaviour to be configured.
- feat: support request validation for OpenAPI specs.

### Changed
- docs: adds validation example project.

## [0.25.2] - 2025-04-09
### Changed
- build(deps): bump golang.org/x/net from 0.33.0 to 0.36.0 (#14)

## [0.25.1] - 2025-03-04
### Fixed
- fix: build system.server.url placeholder from server port.

## [0.25.0] - 2025-03-04
### Added
- feat: add CORS handling support.
- feat: support converting legacy colon-style paths to OpenAPI format.

## [0.24.2] - 2025-03-03
### Fixed
- fix: JSONPath should return nil for nonexistent key.

## [0.24.1] - 2025-02-26
### Fixed
- fix: aligns store API responses to original implementation.

## [0.24.0] - 2025-02-25
### Added
- feat: supports withFile, withDelay, withDelayRange and withFailure in scripts.

## [0.23.2] - 2025-02-22
### Fixed
- fix: legacy root resource shouldn't change resource index.
- fix: resolve openapi spec files relative to config file.

## [0.23.1] - 2025-02-18
### Changed
- refactor: improve efficiency of legacy config transformation.
- refactor: simplify legacy config detection.

### Fixed
- fix: legacy config converter should preserve more properties.

## [0.23.0] - 2025-02-18
### Added
- feat: support legacy withData script function.

### Changed
- refactor: use go style for store handler logging.
- refactor: use logger package in store handler.
- test: adds conformance script.

### Fixed
- fix: legacy config converter should preserve response properties.
- fix: support legacy staticData response property.

## [0.22.1] - 2025-02-14
### Changed
- refactor: removes unneeded creation of in-memory store provider.

## [0.22.0] - 2025-02-14
### Added
- feat: include form params in script request context.
- feat: include path params in script request context.

### Changed
- docs: all step types are supported.
- refactor: moves request store handling into store package.
- refactor: supports 'js' as script step lang.

### Fixed
- fix: path params extraction should use request matcher.

## [0.21.1] - 2025-02-10
### Fixed
- fix: legacy config detector should check for scriptFile.

## [0.21.0] - 2025-02-10
### Added
- feat: script steps mutate response state.
- feat: transforms legacy scriptFile config to steps.

## [0.20.0] - 2025-02-09
### Added
- feat: adds remote step support.
- feat: adds script step support.
- feat: adds steps execution skeleton.
- feat: adds steps model and example.

### Changed
- docs: steps and JS scripts are supported.
- refactor: adds exchange to wrap request context.
- refactor: format model.

## [0.19.1] - 2025-02-07
### Changed
- ci: excludes release commit messages from release notes.
- refactor: moves capture config out of request matcher.
- refactor: moves some packages out of internal.

## [0.19.0] - 2025-02-04
### Added
- feat: switches request matcher regex engine to github.com/dlclark/regexp2.

## [0.18.1] - 2025-02-04
### Changed
- docs: setting example name is supported.

### Fixed
- fix: discount negative matches and prefer non-runtime-generated resources.

## [0.18.0] - 2025-02-03
### Added
- feat(openapi): supports setting response example by name.

### Changed
- refactor(soap): preserve response status, delay and failure if fault set.

## [0.17.5] - 2025-01-31
### Changed
- refactor: response file/content handling shouldn't update config.

## [0.17.4] - 2025-01-31
### Changed
- docs: expand testing instructions.
- refactor: handler passes standard response processor to plugin.

## [0.17.3] - 2025-01-30
### Changed
- refactor(openapi): moves response lookup into operation.

## [0.17.2] - 2025-01-30
### Changed
- refactor(openapi): makes response example generation lazy.
- refactor(soap): makes response example generation lazy.
- refactor: adds response trace logging.

## [0.17.1] - 2025-01-30
### Changed
- refactor(soap): improves logging when failing to parse request.

## [0.17.0] - 2025-01-30
### Added
- feat(openapi): return example headers from spec.

### Changed
- test: improves coverage of openapi utils.

### Fixed
- fix(openapi): improves example coercion.

## [0.16.1] - 2025-01-30
### Changed
- docs: composite and XSD type WSDL messages are supported.
- docs: updates CLI link.

### Fixed
- fix: always prepend security interceptors.

## [0.16.0] - 2025-01-29
### Added
- feat(soap): generate synthetic schema for non-element messages.

### Changed
- test: improves coverage of utils.
- test: moves xpath assertions into table.

### Fixed
- fix(soap): improves namespace prefixing for synthetic schemas.
- fix(soap): synthetic schemas should import dependencies.

## [0.15.4] - 2025-01-23
### Changed
- build: creates placeholder directories in Docker image.

### Fixed
- fix: ignore invalid message parts attribute.

## [0.15.3] - 2025-01-23
### Changed
- build: disables cgo in Docker build.

## [0.15.2] - 2025-01-23
### Changed
- refactor: startup log should print version.

## [0.15.1] - 2025-01-23
### Changed
- docs: adds CLI instructions.
- docs: describes recommended approach.
- docs: fully qualify releases URL.
- docs: improves usage instructions.
- docs: indicate version is required when using the CLI.
- docs: provides example for version argument.
- test: moves model tests to separate file.

### Fixed
- fix: resolve preloaded store file paths relative to config file.

## [0.15.0] - 2025-01-22
### Added
- feat(soap): adds message part filtering.
- feat(soap): adds support for message part parsing.

### Changed
- test(soap): merges SOAP 1.1 and 1.2 fault tests.
- test(soap): run against all 3 supported WSDL/SOAP version combinations.

## [0.14.1] - 2025-01-20
### Changed
- docs: adds more metrics to performance results.
- docs: adds performance test results.
- docs: dir responses are supported.
- refactor: normalises file path before building response.
- refactor: normalises response file path and store preload file path.

## [0.14.0] - 2025-01-20
### Added
- feat: supports directory response and wildcard paths.

### Changed
- docs: links to pending issues.
- docs: updates supported features and adds usage.

## [0.13.1] - 2025-01-19
### Changed
- docs: removes duplicate changelog entry.
- refactor(openapi): improves JSON schema example rendering.
- refactor(openapi): improves object example rendering.

## [0.13.0] - 2025-01-18
### Added
- feat(openapi): gather more media types from swagger examples.
- feat(openapi): implements schema example generator.
- feat(openapi): supports OAS3 server paths and swagger basepath.
- feat: adds schema support for anyOf, oneOf and allOf.
- feat: support multiple config dirs.

### Changed
- docs: removes old schema.
- docs: removes some completed items.
- refactor(openapi): be defensive around schemas/components in specs.
- refactor(soap): improves WSDL parser logging.
- refactor: improves example YAML node marshalling.
- refactor: infer schema type from properties if unset.
- refactor: remove unused script.
- test: improves coverage of OAS2 handler.
- test: improves coverage of OAS3 handler.

## [0.12.1] - 2025-01-17
### Fixed
- fix: Resolve preload file paths relative to config file.

## [0.12.0] - 2025-01-17
### Added
- feat(openapi): adds OAS parser.
- feat(openapi): augment config from OpenAPI spec.
- feat(openapi): prepopulates responses for status codes and matching media types.
- feat(openapi): supports OAS2 and OAS3.
- feat: adds boilerplate for openapi plugin.
- feat: skeleton for OpenAPI parser.

### Changed
- refactor(soap): improves augmentation logging.

## [0.11.1] - 2025-01-16
### Changed
- build: adds fmt target.
- build: don't run tests in verbose mode by default.
- refactor: moves matcher unmarshaller to model.

### Fixed
- fix: HTTP method comparison should fold case.

## [0.11.0] - 2025-01-16
### Added
- feat: supports JSONPath and XPath template queries.

## [0.10.1] - 2025-01-15
### Changed
- refactor(soap): removes unneeded matcher types.

### Fixed
- fix(soap): correct single body matcher unmarshalling.
- fix(soap): removes duplicate file and path prefixing.

## [0.10.0] - 2025-01-15
### Added
- feat(soap): adds support for fault generation.

### Changed
- refactor(soap): removes redundant op lookup.

## [0.9.2] - 2025-01-14
### Changed
- refactor(soap): improves 404 resource list.

### Fixed
- fix(soap): fail if a non-matching SOAPAction is specified in the request.

## [0.9.1] - 2025-01-14
### Fixed
- fix: input element namespace should use schema targetNamespace.

## [0.9.0] - 2025-01-14
### Added
- feat: adds XSD base types to schema type system.

## [0.8.1] - 2025-01-13
### Fixed
- fix: processed schemas should inherit ancestor namespace prefixes.

## [0.8.0] - 2025-01-12
### Added
- feat(soap): improves WSDL message parsing.
- feat: adds XML example generator.
- feat: improves SOAP response example generation.

### Changed
- refactor(soap): moves WSDL parser logic into separate files.

### Fixed
- fix: WSDL 2 parser should look up interface operation from binding ref.

## [0.7.0] - 2025-01-08
### Added
- feat(wsdl): prepopulate responses based on WSDL.
- feat: logs start up time.
- feat: supports more legacy format fields.

### Changed
- docs: splits legacy and current schema files.
- refactor: generalises capture signature to support interceptors.
- refactor: improves failed handler logging.
- refactor: lambda init should be conditional on runtime.
- refactor: moves configuration parsing earlier in lifecycle.

### Fixed
- fix(soap): check all attributes of root element to determine WSDL version.

## [0.6.1] - 2025-01-06
### Changed
- build: removes unsupported install flag.

### Fixed
- fix: request header property should be 'requestHeaders'.

## [0.6.0] - 2025-01-06
### Added
- feat: adds anyOf expression matcher.

### Changed
- docs: adds example for security config.
- refactor: improves expression matcher config naming.
- refactor: switches transformed security interceptors to anyOf expressions.

### Fixed
- fix: improves SOAP version namespace validation.

## [0.5.0] - 2025-01-05
### Added
- feat: transforms security blocks into interceptors.

### Changed
- build: adds since config.
- test: adds delay tolerance to more unit test conditions.

## [0.4.1] - 2025-01-05
### Changed
- test: adds delay tolerance to unit test.

## [0.4.0] - 2025-01-05
### Added
- feat: adds 'evals' matcher.

### Changed
- test: improves coverage for legacy config.
- test: splits matcher tests.

## [0.3.0] - 2025-01-04
### Added
- feat: adds config file schema and validator.
- feat: adds support for legacy config format.

## [0.2.0] - 2025-01-03
### Added
- feat: supports system XML namespaces.

### Changed
- docs: fixes JSONPath and XPath example configs.
- docs: improves release template.

## [0.1.2] - 2025-01-03
### Changed
- build: aligns binary name and build tags.
- docs: fixes example path.
- docs: improves installation instructions.

## [0.1.1] - 2025-01-03
### Changed
- ci: goreleaser should set internal version.

## [0.1.0] - 2025-01-03
### Added
- feat: first release.

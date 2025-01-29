# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

// Package httptestmock provides an HTTP mock server framework for tests.
//
// Features:
//   - Define mocks programmatically with Mock builder or load from JSON/YAML
//   - Request matching by method, path, query parameters, headers, and body
//   - Response templates with status, headers, and body
//   - Custom request handlers for dynamic responses
//   - Hit-count and assertion tracking per mock
//   - Partial and full matching modes
//
// Key types:
//   - Mock: builder for defining a single mock (method, path, response, assertions)
//   - MockHandler: serves registered mocks, tracks requests, validates assertions
//   - RequestMatchLevel: MatchLevelNone, MatchLevelPartial, MatchLevelFull
//   - CustomHandlerFunc: type for dynamic response handlers
//
// Key functions:
//   - NewMock: create a new mock with method and path
//   - SetupServer: create a complete test server with cleanup
//   - GetMocksFrom: load mocks from JSON/YAML files
//   - GetMockHandlerFromServer: extract MockHandler from an httptest.Server
//   - CreateTestRequest: build a test request against the server
//
// See README.md for detailed usage examples.
package httptestmock

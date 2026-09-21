// vitest.setup.ts — global setup for every test file.
//
// Two responsibilities:
//   1. Add @testing-library/jest-dom's custom matchers (toBeInTheDocument,
//      toHaveAttribute, etc.) so React tests get readable assertions.
//   2. Stub nothing else. Each test stubs what it actually needs
//      (cookies via document.cookie, network via fetch mocks) so
//      the test reads as straightforward as the production call site.
import "@testing-library/jest-dom";

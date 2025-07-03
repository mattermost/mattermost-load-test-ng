# Mattermost Browser Load Testing Agent

## Overview

The Browser Load Testing Agent is a specialized component of the Mattermost Load Test Framework that enables browser-based load testing using Playwright. This tool simulates real user interactions with the Mattermost web client, providing valuable insights into the platform's performance under various browser-based workloads.

## Architecture

### Core Components

1. **Server (src/server.ts)**
   - Entry point for the browser agent
   - Configurable through environment variables
   - Handles HTTP server setup and initialization

2. **App (src/app.ts)**
   - Fastify-based application setup
   - Registers routes and middleware
   - Configures logging and server options

3. **Browser Manager (src/lib/browser_manager.ts)**
   - Manages browser sessions using Playwright
   - Handles browser lifecycle (creation, execution, cleanup)
   - Implements session state management
   - Supports parallel test execution

4. **Test Scenarios (src/tests/)**
   - Contains automated browser interactions
   - Implements common user actions:
     - Login
     - Posting messages
     - Channel navigation
     - Scrolling through message history

### Configuration

The project uses TypeScript and is configured with:
- `tsconfig.json` for TypeScript compilation
- `playwright.config.ts` for Playwright test settings
- `prettier.config.cjs` for code formatting
- Environment variables for runtime configuration

## Setup and Usage

### Prerequisites

- Node.js (version specified in `.nvmrc`)
- npm (Node package manager)

### Installation

```bash
cd browser
make install
```

This will:
- Install Node.js dependencies
- Install Playwright Chromium browser with dependencies

For dependencies only (without Playwright):
```bash
make install-dependencies
```

### Running the Server

Production mode:
```bash
make start
```

Development mode:
```bash
make start-dev
```

Watch mode (for development):
```bash
make start-watch
```

Build only:
```bash
make build
```

### Environment Variables

- `BROWSER_AGENT_API_URL`: Required. The URL where the browser agent will listen (e.g., `http://127.0.0.1:5000`)

## Test Scenarios

The browser agent includes both programmatic test scenarios for load testing and standalone test specifications for manual testing and development.

### Test Structure

Each test scenario consists of two main files:

1. **Scenario Implementation** (`scenario1.ts`) - Used by the browser server API for automated load testing
2. **Test Specification** (`scenario1.spec.ts`) - Used for manual testing with Playwright

#### Key Differences:
- **Implementation file**: Runs in infinite loop for continuous load testing, accepts dynamic user credentials
- **Spec file**: Runs once with hardcoded credentials for manual testing and validation

### Running Manual Tests

#### Prerequisites
```bash
cd browser
make install
```

#### Running Individual Test Scenarios

1. **Run all E2E tests**:
   ```bash
   npm run e2etest:run
   ```

2. **Run tests with UI (interactive mode)**:
   ```bash
   npm run e2etest:ui
   ```

3. **Debug tests step by step**:
   ```bash
   npm run e2etest:debug
   ```

4. **Run specific test file**:
   ```bash
   npx playwright test src/simulations/scenario1.spec.ts
   ```

5. **Run tests in headed mode (visible browser)**:
   ```bash
   npx playwright test --headed
   ```

6. **Run unit tests**:
   ```bash
   npm run unittest:run      # Run once
   npm run unittest:watch    # Run in watch mode
   npm run unittest:ui       # Run with UI
   ```

#### Test Configuration

Tests are configured in `playwright.config.ts`:
```typescript
export default defineConfig({
  testDir: './src/simulations',
  outputDir: './e2etest-results',
  fullyParallel: true,
  use: {
    trace: 'off',
  },
  projects: [
    {
      name: 'chromium',
      use: {...devices['Desktop Chrome']},
    },
  ],
});
```

#### Creating New Test Scenarios

To create a new test scenario:

1. **Create the implementation file** (`src/simulations/scenarioN.ts`):
   ```typescript
   export async function scenarioN({page, userId, password}: BrowserInstance, serverURL: string) {
     // Implementation for load testing (infinite loop)
   }
   ```

2. **Create the test specification file** (`src/simulations/scenarioN.spec.ts`):
   ```typescript
   import {test} from '@playwright/test';
   
   test('Scenario N', async ({page}) => {
     // Single-run test for manual validation
   });
   ```

3. **Update the browser manager** to include the new scenario in the test execution.

### Test Results and Reports

- Test results are stored in `e2etest-results/`
- Playwright generates detailed HTML reports
- Screenshots and videos are captured on failures
- Trace files can be generated for debugging

## Development

### Project Structure

```
browser/
├── src/
│   ├── app.ts              # Application setup
│   ├── server.ts           # Server entry point
│   ├── config/             # Configuration management
│   ├── lib/               # Core libraries
│   ├── routes/            # API endpoints
│   └── tests/             # Test scenarios
├── scripts/               # Utility scripts
└── e2etest-results/      # Test results and reports
```

### Testing

- Unit tests: `npm run unittest:run` or `npm run unittest:watch`
- E2E tests: `npm run e2etest:run` or `npm run e2etest:ui`
- Test results are stored in `test-results/` and `e2etest-results/`

### Best Practices

1. **Code Style**
   - Follow TypeScript best practices
   - Use Prettier for code formatting
   - Maintain consistent error handling

2. **Testing**
   - Write unit tests for new features
   - Implement E2E tests for scenarios
   - Use proper mocking and test isolation

3. **Error Handling**
   - Implement proper error boundaries
   - Log errors with appropriate context
   - Clean up resources on failure

## Logging

The project implements comprehensive logging with:
- Console logging for development
- File logging for production
- Configurable log levels
- Structured log format

## Security Considerations

1. **Browser Sessions**
   - Proper cleanup of browser instances
   - Secure handling of credentials
   - Session isolation

2. **API Security**
   - Input validation
   - Rate limiting
   - Proper error handling

## License

Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
See LICENSE.txt for license information.

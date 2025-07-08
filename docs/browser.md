# NOTE: This is a work in progress. Should be reviewed before merging.

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
- Install Playwright Chromium browser with dependencies (headless mode only by default)

For dependencies only (without Playwright):
```bash
make install-dependencies
```

#### Browser Installation Modes

By default, the Makefile installs Playwright in headless-only mode for better performance and smaller footprint. If you need the complete browser installation (including UI components), you can override this behavior:

```bash
# Install with complete browser (including UI)
PLAYWRIGHT_HEADLESS_ONLY=false make install

# Or when running in development mode
PLAYWRIGHT_HEADLESS_ONLY=false make dev
```

**Note**: Complete browser installation is larger and includes additional dependencies, but may be necessary for certain testing scenarios or debugging purposes.

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

### Configuration

The browser agent uses configuration from `config/config.json` in the project root. Key browser-related settings include:

#### BrowserConfiguration
```json
{
  "BrowserConfiguration": {
    "Headless": true
  }
}
```
- `Headless`: Controls whether browsers run in headless mode (true) or with UI (false)

#### BrowserLogSettings
```json
{
  "BrowserLogSettings": {
    "EnableConsole": true,
    "ConsoleLevel": "debug",
    "EnableFile": true,
    "FileLevel": "info",
    "FileLocation": "browseragent.log"
  }
}
```
- `EnableConsole`: Enable/disable console logging
- `ConsoleLevel`: Log level for console output. Possible values (in order of decreasing verbosity): `trace`, `debug`, `info`, `warn`, `error`, `fatal`
- `EnableFile`: Enable/disable file logging
- `FileLevel`: Log level for file output. Same values as `ConsoleLevel`: `trace`, `debug`, `info`, `warn`, `error`, `fatal`
- `FileLocation`: Path to the log file

When both `EnableConsole` and `EnableFile` are true, the logs are written asynchronously to reduce overhead.

#### UsersConfiguration
```json
{
  "UsersConfiguration": {
    "MaxActiveBrowserUsers": 10
  }
}
```
- `MaxActiveBrowserUsers`: Maximum number of concurrent browser sessions

**Important**: After modifying `BrowserLogSettings` in `config/config.json`, you must restart the Browser API server for changes to take effect:

```bash
# Stop the current server (Ctrl+C) then restart
make start        # For production
make start-dev    # For development
```

## Test Scenarios

The browser agent includes both programmatic test scenarios for load testing and standalone test specifications for manual testing and development.

### Test Structure

Each test scenario consists of two main files:

1. **Scenario Implementation** (`src/simulations/scenario_1.ts`) - Used by the browser server API for automated load testing
2. **Test Specification** (`src/e2e/scenario_1.spec.ts`) - Used for manual testing with Playwright

#### Current Implementation

**Scenario Implementation**: `src/simulations/scenario_1.ts`
```typescript
export async function scenario1({page, userId, password}: BrowserInstance, serverURL: string, runInLoop = true) {
  if (!page) {
    throw new Error('Page is not initialized');
  }

  try {
    await page.goto(serverURL);
    await handlePreferenceCheckbox(page);
    await performLogin({page, userId, password});

    // Runs the simulation atleast once and then runs it in continuous loop if simulationMode is true
    do {
      const scrollCount = runInLoop ? 40 : 3;
      await postInChannel({page});
      await scrollInChannel(page, 'sidebarItem_off-topic', scrollCount, 400, 500);
      await scrollInChannel(page, 'sidebarItem_town-square', scrollCount, 400, 500);
    } while (runInLoop);
  } catch (error: any) {
    throw {error: error?.error, testId: error?.testId};
  }
}
```

**Test Specification**: `src/e2e/scenario_1.spec.ts`
```typescript
import {test} from '@playwright/test';
import {scenario1} from '../simulations/scenario_1.js';
import {getMattermostServerURL} from '../utils/config.js';
import type {BrowserInstance} from '../lib/browser_manager.js';

test('Scenario 1', async ({page}) => {
  const browserInstance = {
    page,
    userId: 'user1@example.com',
    password: 'Password-1!',
  } as BrowserInstance;

  const serverURL = getMattermostServerURL();

  await scenario1(browserInstance, serverURL, false);
});
```

#### Key Differences:
- **Implementation file**: Accepts `runInLoop` parameter to control continuous execution for load testing
- **Spec file**: Wraps the simulation function and calls it with `runInLoop: false` for single-run testing
- **Spec file**: Uses configuration from `config.json` for server URL and provides test credentials
- **Spec file**: Compatible with Playwright's test runner and reporting

### Running Manual Tests

#### Prerequisites

For running E2E tests, you need the complete browser installation (not just headless mode):

```bash
cd browser
PLAYWRIGHT_HEADLESS_ONLY=false make install
```

If you already have the headless-only installation, you can install the complete browser:

```bash
PLAYWRIGHT_HEADLESS_ONLY=false make install-playwright
```

**Note**: The complete browser installation is required for E2E tests because they may need UI components and debugging capabilities that are not available in headless-only mode.

#### Running Test Scenarios with Playwright

The project includes npm scripts that handle the proper Node.js configuration for running TypeScript tests:

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
   NODE_OPTIONS='--import=tsx/esm' npx playwright test src/e2e/scenario_1.spec.ts
   ```

5. **Run tests in headed mode (visible browser)**:
   ```bash
   NODE_OPTIONS='--import=tsx/esm' npx playwright test --headed
   ```

6. **Run tests with specific browser**:
   ```bash
   NODE_OPTIONS='--import=tsx/esm' npx playwright test --project=chromium
   ```

#### Running Unit Tests

7. **Run unit tests**:
   ```bash
   npm run unittest:run      # Run once
   npm run unittest:watch    # Run in watch mode
   npm run unittest:ui       # Run with UI
   ```

#### Important Notes

- The project uses `NODE_OPTIONS='--import=tsx/esm'` to handle TypeScript imports properly
- All npm scripts in `package.json` include the necessary Node.js configuration
- When running Playwright commands directly (not through npm scripts), you must include the `NODE_OPTIONS` environment variable
- Test files are located in `src/e2e/` directory for E2E tests and throughout `src/` for unit tests

#### Troubleshooting E2E Tests

**1. Server URL Configuration Issues**
- **Problem**: Tests fail with connection errors or navigation timeouts
- **Solution**: Check if the server URL is present and correct in `config/config.json`:
  ```json
  {
    "ConnectionConfiguration": {
      "ServerURL": "http://localhost:8065"
    }
  }
  ```

**2. Missing Dependencies in UI Mode**
- **Problem**: Playwright UI mode fails to start or shows errors
- **Solution**: If opening tests in UI mode, Playwright might need to install additional dependencies. Check "Toggle output" on the top left of the Playwright runner to see if there are any missing dependencies. Install them with:
  ```bash
  PLAYWRIGHT_HEADLESS_ONLY=false make install-playwright
  ```

**3. Browser Launch Failures**
- **Problem**: Tests fail with "Browser not found" or launch errors
- **Solution**: Ensure you have the complete browser installation:
  ```bash
  PLAYWRIGHT_HEADLESS_ONLY=false make install
  ```
  If the issue persists, try clearing the browser cache:
  ```bash
  npx playwright install --force chromium
  ```

**4. TypeScript Import Errors**
- **Problem**: Tests fail with module import errors or TypeScript compilation issues
- **Solution**: Ensure you're using the correct Node.js options. Always use the npm scripts or include `NODE_OPTIONS='--import=tsx/esm'` when running Playwright directly:
  ```bash
  NODE_OPTIONS='--import=tsx/esm' npx playwright test
  ```

**5. Test Timeout Issues**
- **Problem**: Tests timeout during page navigation or element interactions
- **Solution**: 
  - Verify the target Mattermost server is running and accessible
  - Check network connectivity to the server
  - Increase timeout in `playwright.config.ts` if needed:
    ```typescript
    use: {
      actionTimeout: 30000,
      navigationTimeout: 30000,
    }
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

1. **Create the implementation file** (`src/simulations/scenario_N.ts`):
   ```typescript
   import type {BrowserInstance} from '../lib/browser_manager.js';
   
   export async function scenarioN({page, userId, password}: BrowserInstance, serverURL: string, runInLoop = true) {
     if (!page) {
       throw new Error('Page is not initialized');
     }
   
     try {
       await page.goto(serverURL);
       // Add your scenario logic here
       
       do {
         // Your test actions here
         // Adjust behavior based on runInLoop parameter
       } while (runInLoop);
     } catch (error: any) {
       throw {error: error?.error, testId: error?.testId};
     }
   }
   ```

2. **Create the test specification file** (`src/e2e/scenario_N.spec.ts`):
   ```typescript
   import {test} from '@playwright/test';
   import {scenarioN} from '../simulations/scenario_N.js';
   import {getMattermostServerURL} from '../utils/config.js';
   import type {BrowserInstance} from '../lib/browser_manager.js';
   
   test('Scenario N', async ({page}) => {
     const browserInstance = {
       page,
       userId: 'testuser@example.com',
       password: 'TestPassword123!',
     } as BrowserInstance;
   
     const serverURL = getMattermostServerURL();
   
     await scenarioN(browserInstance, serverURL, false);
   });
   ```

3. **Update the browser manager** to include the new scenario in the test execution.

#### Benefits of This Pattern

- **Reusability**: Same simulation code works for both load testing and manual testing
- **Consistency**: Ensures manual tests use the same logic as load tests
- **Flexibility**: `runInLoop` parameter allows different behaviors for different contexts
- **Configuration**: Spec files can use centralized configuration while simulations remain flexible
- **Playwright Integration**: Spec files work seamlessly with Playwright's test runner and reporting

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

// Set environment variable to indicate local execution
process.env.IS_LOCAL = 'true';

const { handler } = require('./index');
const fs = require('fs');

function parseArgs() {
  const args = process.argv.slice(2);
  const params = {
    debug: false,
    reload: false,
    delay: 5000
  };

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];

    if (arg === '--username' || arg === '-u') {
      params.username = args[++i];
    } else if (arg === '--password' || arg === '-p') {
      params.password = args[++i];
    } else if (arg === '--url' || arg === '-l') {
      params.url = args[++i];
    } else if (arg === '--debug' || arg === '-d') {
      params.debug = true;
    } else if (arg === '--reload' || arg === '-r') {
      params.reload = true;
    } else if (arg === '--delay') {
      params.delay = parseInt(args[++i], 10);
    } else if (arg === '--join_team') {
      params.join_team = args[++i];
    } else if (arg === '--help' || arg === '-h') {
      showHelp();
      process.exit(0);
    }
  }

  // Validate required parameters
  if (!params.username || !params.password || !params.url) {
    console.error('Error: Username, password, and url are required');
    showHelp();
    process.exit(1);
  }

  return params;
}

function showHelp() {
  console.log(`
Usage: node local-test.js [options]

Options:
  -u, --username <username>  Mattermost username (required)
  -p, --password <password>  Mattermost password (required)
  -l, --url <url>            Mattermost URL (required)
  -d, --debug                Enable debug mode with screenshots (default: false)
  -r, --reload               Reload the page after delay (default: false)
  --delay <ms>               Delay after login in milliseconds (default: 5000)
  --join_team <team_id>      Join the given team_id after logging in (default: <empty>)
  -h, --help                 Show this help message
  `);
}

async function runLocalTest() {
  // If help is requested, show help and exit
  if (process.argv.includes('--help') || process.argv.includes('-h')) {
    showHelp();
    return;
  }

  const testEvent = parseArgs();

  console.log(`Starting local test: ${testEvent.url}`);

  const result = await handler(testEvent, {});

  // Save the result to a file for inspection
  fs.writeFileSync('response-local.json', JSON.stringify(result, null, 2));
  console.log('Response saved to response-local.json');

  // Parse the body once
  let body = {};
  if (result.body) {
    body = JSON.parse(result.body);

    // Save screenshot if available
    if (body.screenshot) {
      fs.writeFileSync('screenshot.png', Buffer.from(body.screenshot, 'base64'));
      console.log('Screenshot saved to screenshot.png');
    }
  }

  console.log('Test completed with status:', result.statusCode, 'Message:', body.message);
  return result;
}

runLocalTest().catch(error => {
  console.error('Test failed with error:', error);
});

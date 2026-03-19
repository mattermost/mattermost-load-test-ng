# Mattermost load-test Lambda function

This is a quick demo using a Lambda function to load-test some of the edge-cases that our primary tool is unable to do. For example:
- Websocket reconnect.
- Large number of users joining a team in a short time.
- Large number of users joining a channel in a short time.
- Possible others.

## Required Libraries and Dependencies

- `puppeteer-core`: Provides the API to control Chromium.
- The `@sparticuz/chromium` library is provided as a layer.
- Node.js 14 or higher recommended

## Installation

First, use this layer in your function : `arn:aws:lambda:us-east-1:764866452798:layer:chrome-aws-lambda:50`

Then,
```bash
npm install
Zip the entire project: zip -r deployment.zip index.js package.json node_modules
Upload the code: aws lambda update-function-code --function-name <functionName> --zip-file fileb://deployment.zip
Invoke the function: aws lambda invoke --function-name <functionName> --cli-binary-format raw-in-base64-out --payload '{"username": "<username>", "password": "<password>", "url": <url>}' response.json
```

## Testing locally:

```
node local-test.js --url "<>" -u <> -p <> --delay 5000
```

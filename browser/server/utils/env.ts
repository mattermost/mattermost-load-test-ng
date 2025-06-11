import path from 'path';
import {fileURLToPath} from 'url';
import dotenv from 'dotenv';

/**
 * Load environment variables from .env file
 */
export default function loadEnv() {
  const dirname = path.dirname(fileURLToPath(import.meta.url));
  const envPath = path.resolve(dirname, '../.env');
  const dotenvConfig = dotenv.config({path: envPath});

  if (dotenvConfig.error) {
    console.error('Failed to load .env file:', dotenvConfig.error);
    process.exit(1);
  } else {
    console.log('Loaded .env file successfully');
  }
}

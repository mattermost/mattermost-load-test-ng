#!/usr/bin/env node

import { build } from 'esbuild';
import { devConfig, prodConfig } from '../esbuild.config.js';

const isDev = process.env.NODE_ENV === 'development';
const isWatch = process.argv.includes('--watch');

const config = isDev ? devConfig : prodConfig;

// Add watch mode if requested
if (isWatch) {
  config.watch = {
    onRebuild(error, result) {
      if (error) {
        console.error('Watch build failed:', error);
      } else {
        console.log('Watch build succeeded');
      }
    },
  };
}

async function runBuild() {
  try {
    console.log(`Building in ${isDev ? 'development' : 'production'} mode...`);

    if (isWatch) {
      console.log('Watching for changes...');
    }

    const result = await build(config);

    if (!isWatch) {
      console.log('Build completed successfully!');
      console.log(`Output: ${config.outfile}`);
    }

    return result;
  } catch (error) {
    console.error('Build failed:', error);
    process.exit(1);
  }
}

runBuild();

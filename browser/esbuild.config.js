import { build } from 'esbuild';
import { readFileSync } from 'fs';

// Read package.json to get version and other metadata
const packageJson = JSON.parse(readFileSync('./package.json', 'utf8'));

const baseConfig = {
  entryPoints: ['src/server.ts'],
  bundle: true,
  platform: 'node',
  target: 'node22',
  format: 'esm',
  outfile: 'dist/server.js',
  external: [
    // Platform-specific dependencies that should remain external
    'playwright',
    '@playwright/test',
    // Node.js built-ins (optional, esbuild handles these automatically)
    'fs',
    'path',
    'http',
    'https',
    'url',
    'os'
  ],
  define: {
    // Inject version from package.json
    'process.env.VERSION': JSON.stringify(packageJson.version),
    'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV || 'production')
  },
  banner: {
    js: '#!/usr/bin/env node'
  },
  // Tree shaking and optimization
  treeShaking: true,
  // Source maps for debugging
  sourcemap: true,
  // Minification for production
  minify: process.env.NODE_ENV === 'production',
  // Keep names for better debugging
  keepNames: true,
  // Handle JSON imports
  loader: {
    '.json': 'json'
  }
};

// Development configuration
const devConfig = {
  ...baseConfig,
  minify: false,
  sourcemap: 'inline',
  watch: process.argv.includes('--watch')
};

// Production configuration
const prodConfig = {
  ...baseConfig,
  minify: true,
  sourcemap: true,
  drop: ['console', 'debugger'] // Remove console.log and debugger statements
};

// Export configurations
export { baseConfig, devConfig, prodConfig };

// If this file is run directly, build based on NODE_ENV
if (import.meta.url === `file://${process.argv[1]}`) {
  const config = process.env.NODE_ENV === 'development' ? devConfig : prodConfig;

  build(config).catch((error) => {
    console.error('Build failed:', error);
    process.exit(1);
  });
}

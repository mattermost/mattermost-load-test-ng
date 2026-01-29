// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import * as esbuild from 'esbuild';

const OutputFolder = 'dist';
const OutFile = `${OutputFolder}/index.js`;

async function generateBundle() {
    await esbuild.build({
        entryPoints: ['src/index.ts'],
        bundle: true,
        platform: 'node',
        target: 'node24',
        minify: false,
        format: 'esm',
        outfile: OutFile,
        packages: 'bundle',
        external: [
            '@mattermost/playwright-lib',
            '@playwright/test',
        ],
    });

    console.log('@mattermost/loadtest-browser-lib: Bundle generated');
}

await generateBundle();

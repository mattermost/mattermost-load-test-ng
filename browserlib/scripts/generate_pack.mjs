// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {execSync} from 'child_process';
import {renameSync, unlinkSync, existsSync} from 'fs';
import path from 'path';

const OutputDir = '../browser/packs';
const OutputFileName = 'loadtest-browser-lib.tgz';
const OutputFile = path.join(OutputDir, OutputFileName);

// Remove existing file if it exists
if (existsSync(OutputFile)) {
    unlinkSync(OutputFile);
}

const packOutput = execSync(`npm pack --json --silent --pack-destination ${OutputDir}`, {encoding: 'utf-8'}).trim();

// Since the out
const jsonStart = packOutput.indexOf('[');
const jsonEnd = packOutput.lastIndexOf(']');
if (jsonStart === -1 || jsonEnd === -1 || jsonEnd < jsonStart) {
    throw new Error(`npm pack output did not include JSON: ${packOutput}`);
}
const packFilename = JSON.parse(packOutput.slice(jsonStart, jsonEnd + 1))[0].filename;

// Rename to consistent filename
renameSync(path.join(OutputDir, packFilename), OutputFile);

// eslint-disable-next-line no-undef
console.log(`@mattermost/loadtest-browser-lib: Packed to ${OutputFile}`);

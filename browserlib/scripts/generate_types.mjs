// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import ts from 'typescript';
import path from 'path';
import {fileURLToPath} from 'url';

const OutputFolder = 'dist';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const rootDir = path.resolve(__dirname, '..');

function generateTypes() {
    const configPath = path.join(rootDir, 'tsconfig.json');
    const configFile = ts.readConfigFile(configPath, ts.sys.readFile);
    const parsedConfig = ts.parseJsonConfigFileContent(configFile.config, ts.sys, rootDir);

    const program = ts.createProgram(parsedConfig.fileNames, {
        ...parsedConfig.options,
        declaration: true,
        declarationMap: true,
        emitDeclarationOnly: true,
        outDir: path.join(rootDir, OutputFolder),
    });

    const emitResult = program.emit();
    const diagnostics = ts.getPreEmitDiagnostics(program).concat(emitResult.diagnostics);

    if (diagnostics.length > 0) {
        const formatHost = {
            getCanonicalFileName: (fileName) => fileName,
            getCurrentDirectory: ts.sys.getCurrentDirectory,
            getNewLine: () => ts.sys.newLine,
        };
        // eslint-disable-next-line no-undef
        console.error(ts.formatDiagnosticsWithColorAndContext(diagnostics, formatHost));
        // eslint-disable-next-line no-undef
        process.exit(1);
    }

    // eslint-disable-next-line no-undef
    console.log('@mattermost/loadtest-browser-lib: Type declarations generated');
}

generateTypes();

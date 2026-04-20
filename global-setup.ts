/**
 * Copyright 2026 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import {execSync} from 'child_process';
import fs from 'fs';
import path from 'path';

async function globalSetup() {
  if (process.env.USE_DOCKER_BROWSER === 'true') {
    console.log(
      'USE_DOCKER_BROWSER is true. Setting up Docker browser server...',
    );

    const port = process.env.PLAYWRIGHT_DOCKER_PORT || '4444';

    // Read version from package.json
    const pkgPath = path.resolve(process.cwd(), 'package.json');
    const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf-8'));
    const playwrightVersion = pkg.devDependencies['@playwright/test'].replace(
      '^',
      '',
    );

    console.log(`Detected Playwright version: ${playwrightVersion}`);

    // Check if image exists
    try {
      execSync('docker image inspect webstatus-playwright', {stdio: 'ignore'});
      console.log('Docker image webstatus-playwright already exists.');
    } catch (e) {
      console.log(
        'Docker image webstatus-playwright not found. Building it...',
      );
      execSync(
        'docker build -f images/playwright.Dockerfile -t webstatus-playwright .',
        {stdio: 'inherit'},
      );
    }

    // Check if container is already running
    try {
      const existing = execSync('docker ps -q --filter name=playwright-server')
        .toString()
        .trim();
      if (existing) {
        console.log('Playwright server container is already running.');
        process.env.PW_TEST_CONNECT_WS_ENDPOINT = `ws://localhost:${port}`;
        return;
      }
    } catch (e) {
      // Ignore error
    }

    // Start container
    console.log('Starting Playwright server container...');
    const runCmd = `docker run -d --name playwright-server --network host webstatus-playwright bash -c "npx playwright@${playwrightVersion} install --with-deps && npx playwright@${playwrightVersion} run-server --port ${port}"`;

    execSync(runCmd, {stdio: 'inherit'});

    // Wait for port to be ready
    const timeout = process.env.PLAYWRIGHT_DOCKER_TIMEOUT
      ? parseInt(process.env.PLAYWRIGHT_DOCKER_TIMEOUT, 10)
      : 120000;
    console.log(
      `Waiting for Playwright server to be ready on port ${port} (timeout: ${timeout}ms)...`,
    );
    await waitForPort(parseInt(port, 10), timeout);

    process.env.PW_TEST_CONNECT_WS_ENDPOINT = `ws://localhost:${port}`;
    console.log('Playwright server is ready!');
  }
}

async function waitForPort(port: number, timeout = 120000) {
  const start = Date.now();
  while (Date.now() - start < timeout) {
    try {
      const net = await import('net');
      await new Promise((resolve, reject) => {
        const socket = net.connect(port, 'localhost', () => {
          socket.end();
          resolve(true);
        });
        socket.on('error', reject);
      });
      return;
    } catch (e) {
      await new Promise(resolve => setTimeout(resolve, 1000));
    }
  }
  throw new Error(`Timeout waiting for port ${port}`);
}

export default globalSetup;

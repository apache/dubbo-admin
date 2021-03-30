/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
/* eslint-disable eslint-comments/disable-enable-pair */
/* eslint-disable @typescript-eslint/no-var-requires */
/* eslint-disable eslint-comments/no-unlimited-disable */
const { spawn } = require('child_process');
// eslint-disable-next-line import/no-extraneous-dependencies
const { kill } = require('cross-port-killer');

const env = Object.create(process.env);
env.BROWSER = 'none';
env.TEST = true;
env.UMI_UI = 'none';
env.PROGRESS = 'none';
// flag to prevent multiple test
let once = false;

const startServer = spawn(/^win/.test(process.platform) ? 'npm.cmd' : 'npm', ['start'], {
  env,
});

startServer.stderr.on('data', (data) => {
  // eslint-disable-next-line
  console.log(data.toString());
});

startServer.on('exit', () => {
  kill(process.env.PORT || 8000);
});

console.log('Starting development server for e2e tests...');
startServer.stdout.on('data', (data) => {
  console.log(data.toString());
  // hack code , wait umi
  if (
    (!once && data.toString().indexOf('Compiled successfully') >= 0) ||
    data.toString().indexOf('Theme generated successfully') >= 0
  ) {
    // eslint-disable-next-line
    once = true;
    console.log('Development server is started, ready to run tests.');
    const testCmd = spawn(
      /^win/.test(process.platform) ? 'npm.cmd' : 'npm',
      ['test', '--', '--maxWorkers=1', '--runInBand'],
      {
        stdio: 'inherit',
      },
    );
    testCmd.on('exit', (code) => {
      startServer.kill();
      process.exit(code);
    });
  }
});

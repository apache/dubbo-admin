/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 */
import { spawn } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import { setTimeout } from 'node:timers/promises'

const ui = fileURLToPath(new URL('../', import.meta.url))
const children = []
function start(command, args, cwd = ui, env = {}) {
  const childEnvironment = { ...process.env, ...env }
  // Desktop editor environments may export this flag; Cypress needs Electron.
  delete childEnvironment.ELECTRON_RUN_AS_NODE
  const child = spawn(command, args, {
    cwd,
    env: childEnvironment,
    stdio: 'inherit',
    detached: true
  })
  children.push(child)
  child.on('error', (error) => {
    console.error(error)
    process.exitCode = 1
  })
  return child
}
function stop() {
  for (const child of children) {
    try {
      process.kill(-child.pid, 'SIGTERM')
    } catch {
      /* already exited */
    }
  }
}
process.on('SIGINT', () => {
  stop()
  process.exit(130)
})
process.on('SIGTERM', () => {
  stop()
  process.exit(143)
})
async function ready(url, expected, child) {
  for (let attempt = 0; attempt < 240; attempt++) {
    if (child.exitCode !== null) throw new Error(`Fixture exited before readiness: ${url}`)
    try {
      const response = await fetch(url)
      if (response.status === expected) return
    } catch {
      /* starting */
    }
    await setTimeout(1000)
  }
  throw new Error(`Fixture readiness timed out: ${url}`)
}
try {
  for (const port of [8880, 8881]) {
    try {
      await fetch(`http://127.0.0.1:${port}`, { signal: AbortSignal.timeout(1000) })
      throw new Error(`Port ${port} is already in use; stop the existing server first`)
    } catch (error) {
      if (error.message?.includes('already in use')) throw error
    }
  }
  const backend = start(
    'go',
    [
      'test',
      '-count=1',
      '-timeout=15m',
      '-run=^TestMVPBrowserServer$',
      '-v',
      './component/agent/react'
    ],
    fileURLToPath(new URL('../../ai/', import.meta.url)),
    { AI_MVP_BROWSER: '1' }
  )
  await ready('http://127.0.0.1:8880/api/v1/ai/__test/login?user=alice', 204, backend)
  const vite = start(
    process.execPath,
    ['node_modules/vite/bin/vite.js', '--host', '127.0.0.1', '--port', '8881', '--strictPort'],
    ui,
    // Route /api/v1/auth to the fixture, which serves Admin's token endpoints.
    { AI_MVP_BROWSER: '1' }
  )
  await ready('http://127.0.0.1:8881/admin/e2e/mvp.html', 200, vite)
  const cypress = start(process.execPath, [
    'node_modules/cypress/bin/cypress',
    'run',
    '--config',
    'baseUrl=http://127.0.0.1:8881,video=false',
    '--spec',
    'cypress/e2e/memory-mvp.cy.ts'
  ])
  process.exitCode = await new Promise((resolve) =>
    cypress.on('exit', (code) => resolve(code ?? 1))
  )
} catch (error) {
  console.error(error)
  process.exitCode = 1
} finally {
  try {
    // A failed port preflight must not stop somebody else's existing fixture.
    if (children.length > 0) {
      const login = await fetch('http://127.0.0.1:8880/api/v1/ai/__test/login?user=alice')
      const cookie = login.headers.get('set-cookie')?.split(';')[0]
      if (cookie)
        await fetch('http://127.0.0.1:8880/api/v1/ai/__test/stop', {
          method: 'POST',
          headers: { cookie }
        })
    }
  } catch {
    /* backend already stopped */
  }
  stop()
}

import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const dashboard = readFileSync('frontend/src/views/DashboardView.vue', 'utf8');
const statusBar = readFileSync('frontend/src/components/AppStatusBar.vue', 'utf8');
const serverStore = readFileSync('frontend/src/stores/server.ts', 'utf8');
const serverApi = readFileSync('frontend/src/api/server.ts', 'utf8');
const endpoints = readFileSync('frontend/src/api/endpoints.ts', 'utf8');

test('panel update endpoint is wired through the API layer', () => {
  assert.match(endpoints, /checkUpdate: 'panel\/api\/server\/checkUpdate'/);
  assert.match(serverApi, /export function checkPanelUpdate/);
  assert.match(serverApi, /legacyEndpoints\.server\.checkUpdate/);
  assert.match(serverStore, /checkPanelUpdate as fetchPanelUpdate/);
  assert.match(serverStore, /refreshPanelUpdate/);
});

test('dashboard shows a release link when an update is available', () => {
  assert.match(dashboard, /v-if="panelUpdate\?\.updateAvailable"/);
  assert.match(dashboard, /Panel update available/);
  assert.match(dashboard, /:href="panelUpdate\.releaseUrl"/);
  assert.match(dashboard, /refreshPanelUpdate/);
});

test('status bar exposes the update badge', () => {
  assert.match(statusBar, /status-tag-update/);
  assert.match(statusBar, /panelUpdate\?\.updateAvailable/);
  assert.match(statusBar, /panelUpdate\.releaseUrl/);
  assert.match(statusBar, /refreshPanelUpdate/);
});

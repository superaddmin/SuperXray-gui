import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const layoutSource = readFileSync('frontend/src/layouts/MainLayout.vue', 'utf8');
const statusBarSource = readFileSync('frontend/src/components/AppStatusBar.vue', 'utf8');

test('main layout exposes mobile drawer navigation without removing desktop sider', () => {
  assert.match(layoutSource, /Drawer as ADrawer/);
  assert.match(layoutSource, /v-model:open="mobileNavOpen"/);
  assert.match(layoutSource, /class="mobile-nav-drawer"/);
  assert.match(layoutSource, /@breakpoint="handleSiderBreakpoint"/);
  assert.match(layoutSource, /function closeMobileNav/);
  assert.match(layoutSource, /function handleSiderBreakpoint/);
});

test('language toggle accessible name keeps visible label and hidden action text', () => {
  assert.match(statusBarSource, /class="language-toggle"/);
  assert.match(statusBarSource, /:title="languageToggleAriaLabel"/);
  assert.match(statusBarSource, /<span>\{\{ languageButtonLabel \}\}<\/span>/);
  assert.match(statusBarSource, /class="visually-hidden"/);
});

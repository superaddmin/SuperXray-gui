import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const css = readFileSync('frontend/src/styles/app.css', 'utf8');

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

test('form styles define shared section and responsive modal primitives', () => {
  for (const selector of [
    '.form-section',
    '.form-section-header',
    '.form-section-title',
    '.form-section-description',
    '.form-actions',
    '.form-json-stack',
    '.responsive-modal-form',
    '.form-grid--three',
  ]) {
    assert.match(css, new RegExp(escapeRegExp(selector)));
  }
});

test('form styles collapse grids and actions on mobile', () => {
  assert.match(css, /@media \(max-width: 768px\)[\s\S]*\.responsive-modal-form/);
  assert.match(css, /@media \(max-width: 768px\)[\s\S]*\.form-section-header/);
  assert.match(css, /@media \(max-width: 768px\)[\s\S]*\.form-actions/);
  assert.match(css, /@media \(max-width: 768px\)[\s\S]*\.form-grid--three/);
});

test('digital banking theme reserves green for success instead of primary brand', () => {
  assert.match(css, /--brand-bg:\s*#0a1628;/);
  assert.match(css, /--brand-primary:\s*#0066ff;/);
  assert.match(css, /--brand-primary-deep:\s*#0052cc;/);
  assert.match(css, /--brand-success:\s*#36b37e;/);
  assert.doesNotMatch(css, /--brand-primary:\s*#39ff14;/);
});

test('mobile layout hides persistent sider and gives content the viewport', () => {
  assert.match(
    css,
    /@media \(max-width: 760px\)[\s\S]*\.app-sider[\s\S]*transform:\s*translateX\(-100%\)/,
  );
  assert.match(css, /@media \(max-width: 760px\)[\s\S]*\.mobile-nav-drawer/);
  assert.match(css, /@media \(max-width: 760px\)[\s\S]*\.app-content[\s\S]*padding:\s*18px 16px/);
});

test('operational cards use compact radius and stable mobile action grids', () => {
  assert.match(css, /\.status-tile,\s*[\s\S]*\.work-panel[\s\S]*border-radius:\s*8px !important/);
  assert.match(css, /\.page-header-actions--compact/);
  assert.match(css, /@media \(max-width: 760px\)[\s\S]*\.page-header-actions--compact/);
});

test('header glass token overrides ant layout default background', () => {
  assert.match(css, /\.app-header\.ant-layout-header/);
  assert.match(css, /\.app-header\.ant-layout-header[\s\S]*background:\s*rgba\(6, 17, 31, 0\.84\) !important/);
});

test('mobile drawer and compact toggles keep accessible touch targets', () => {
  assert.match(css, /\.drawer-close-button/);
  assert.match(css, /\.ant-switch[\s\S]*min-width:\s*44px/);
  assert.match(css, /\.ant-checkbox-wrapper[\s\S]*min-height:\s*40px/);
  assert.match(css, /\.xray-workspace-nav/);
});

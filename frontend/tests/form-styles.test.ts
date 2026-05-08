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

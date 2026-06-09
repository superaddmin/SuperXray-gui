import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

test('frontend test script includes root legacy asset tests', () => {
  const packageJson = JSON.parse(readFileSync('frontend/package.json', 'utf8')) as {
    scripts?: Record<string, string>;
  };

  assert.match(packageJson.scripts?.test || '', /web\/assets\/js\/\*\.test\.js/);
});

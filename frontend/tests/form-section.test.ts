import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const source = readFileSync('frontend/src/components/FormSection.vue', 'utf8');

test('form section is presentational and slot based', () => {
  assert.match(
    source,
    /defineProps<\{\s*eyebrow\?: string;\s*title: string;\s*description\?: string;\s*\}>/,
  );
  assert.match(source, /<slot name="actions" \/>/);
  assert.match(source, /<slot \/>/);
  assert.match(source, /class="form-section"/);
  assert.doesNotMatch(source, /from '@\/api\//);
  assert.doesNotMatch(source, /submitInbound|saveSettings|restartPanel|importDatabase/);
});

import assert from 'node:assert/strict';
import test from 'node:test';

import { copyText } from '../src/utils/textExport.ts';

test('clipboard fallback removes the selection on success, denial, and exceptions', async (t) => {
  let attached = false;
  let copy: () => boolean = () => true;
  const textarea = {
    style: {},
    select() {},
    setAttribute() {},
    remove() {
      attached = false;
    },
  };
  const documentStub = {
    createElement: () => textarea,
    body: {
      appendChild() {
        attached = true;
      },
    },
    execCommand: () => copy(),
  };
  Object.defineProperty(globalThis, 'document', { configurable: true, value: documentStub });
  t.after(() => Reflect.deleteProperty(globalThis, 'document'));

  await copyText('example');
  assert.equal(attached, false);
  copy = () => false;
  await assert.rejects(copyText('example'), /denied/);
  assert.equal(attached, false);
  copy = () => {
    throw new Error('clipboard unavailable');
  };
  await assert.rejects(copyText('example'), /clipboard unavailable/);
  assert.equal(attached, false);
});

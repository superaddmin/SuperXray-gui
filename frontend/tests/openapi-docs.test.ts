import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import test from 'node:test';

const packageJson = JSON.parse(readFileSync('frontend/package.json', 'utf8')) as {
  dependencies?: Record<string, string>;
  devDependencies?: Record<string, string>;
  scripts?: Record<string, string>;
};
const legacyGeneratedTypeImport = new RegExp(['types', 'generated', 'openapi'].join('\\\\/'));
const routerSource = readFileSync('frontend/src/router/index.ts', 'utf8');
const layoutSource = readFileSync('frontend/src/layouts/MainLayout.vue', 'utf8');
const messagesSource = readFileSync('frontend/src/i18n/messages.ts', 'utf8');
const domTranslatorSource = readFileSync('frontend/src/i18n/domTranslator.ts', 'utf8');
const styleSource = readFileSync('frontend/src/styles/app.css', 'utf8');

test('frontend exposes a Vue-owned API docs route and menu entry', () => {
  assert.match(routerSource, /path:\s*'docs'/);
  assert.match(routerSource, /name:\s*'docs'/);
  assert.match(routerSource, /ApiDocsView\.vue/);
  assert.match(layoutSource, /key:\s*'docs'/);
  assert.match(layoutSource, /nav\.docs/);
});

test('frontend build regenerates local OpenAPI JSON without React or Swagger UI dependencies', () => {
  assert.equal(packageJson.scripts?.['gen:openapi'], 'cd .. && go run ./tools/openapiexport');
  assert.match(packageJson.scripts?.build || '', /npm run gen:openapi/);
  assert.doesNotMatch(packageJson.scripts?.build || '', /api:types|openapi-typescript/);

  const dependencies = JSON.stringify({
    dependencies: packageJson.dependencies,
    devDependencies: packageJson.devDependencies,
  });
  assert.doesNotMatch(
    dependencies,
    /swagger-ui-react|@vitejs\/plugin-react|react-dom|"\s*react"\s*:/,
  );
});

test('OpenAPI loader uses runtime base path and session credentials', () => {
  const source = readFileSync('frontend/src/api/openapi.ts', 'utf8');
  assert.match(source, /getRuntimeConfig/);
  assert.match(source, /panel\/api\/openapi\.json/);
  assert.match(source, /credentials:\s*'include'/);
  assert.match(source, /sessionExpired\s*=\s*status === 404/);
  assert.match(source, /window\.location\.assign\(loginPath\)/);
});

test('API docs view fetches same-origin OpenAPI JSON without HTML sinks and resolves references', () => {
  const viewSource = readFileSync('frontend/src/views/ApiDocsView.vue', 'utf8');
  assert.match(viewSource, /fetchOpenAPIDocument/);
  assert.doesNotMatch(viewSource, /v-html|innerHTML|insertAdjacentHTML/);
  assert.doesNotMatch(viewSource, legacyGeneratedTypeImport);
  assert.doesNotMatch(viewSource, /does not fetch a runtime OpenAPI JSON document/);
  assert.match(viewSource, /function resolveParameter/);
  assert.match(viewSource, /function resolveReference/);
  assert.match(viewSource, /function resolveOpenAPIValue/);
  assert.match(viewSource, /requestBody:\s*resolveOpenAPIValue\(operation\.requestBody\)/);
  assert.match(viewSource, /responses:\s*resolveOpenAPIValue\(operation\.responses\s*\|\|\s*\{\}\)/);
  assert.match(viewSource, /document\.value = undefined/);
});

test('API docs page has bilingual copy, a11y labels, and protects code blocks from DOM translation', () => {
  for (const key of [
    'apiDocs.title',
    'apiDocs.description',
    'apiDocs.searchPlaceholder',
    'apiDocs.searchLabel',
    'apiDocs.methodFilter',
    'apiDocs.tagFilter',
    'apiDocs.paramName',
    'apiDocs.paramRequired',
    'apiDocs.sessionExpired',
    'apiDocs.requestBody',
    'apiDocs.rawOperation',
    'apiDocs.copied',
  ]) {
    assert.match(messagesSource, new RegExp(`'${key}'`));
  }
  assert.match(domTranslatorSource, /\.api-docs-code/);
  assert.match(styleSource, /\.api-docs-page/);
  assert.match(styleSource, /\.api-docs-code/);
  assert.doesNotMatch(styleSource, /\.contract-docs-page|\.contract-docs-list/);
});

test('generated OpenAPI JSON is available as Vite public input', () => {
  assert.equal(existsSync('frontend/public/openapi.json'), true);
  const document = JSON.parse(readFileSync('frontend/public/openapi.json', 'utf8')) as {
    openapi?: string;
    paths?: Record<string, unknown>;
  };
  assert.equal(document.openapi, '3.1.0');
  assert.ok(document.paths?.['/panel/api/openapi.json']);
});

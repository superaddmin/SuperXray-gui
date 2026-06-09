import type { AppLocale } from './messages';
import { translateDomText } from './messages';

const translatableAttributes = ['aria-label', 'alt', 'placeholder', 'title'] as const;
const skippedSelector =
  'script, style, pre, code, textarea, .log-viewport, .code-preview, .api-docs-code, .json-editor';

export function applyLocaleToDocument(locale: AppLocale, root: ParentNode = document.body) {
  document.documentElement.lang = locale;
  translateTextNodes(root, locale);
  translateAttributes(root, locale);
}

export function createLocaleDomObserver(getLocale: () => AppLocale): MutationObserver | undefined {
  if (typeof MutationObserver === 'undefined' || !document.body) {
    return undefined;
  }

  let scheduled = false;
  const observer = new MutationObserver(() => {
    if (scheduled) {
      return;
    }
    scheduled = true;
    window.queueMicrotask(() => {
      scheduled = false;
      applyLocaleToDocument(getLocale());
    });
  });

  observer.observe(document.body, {
    attributeFilter: [...translatableAttributes],
    attributes: true,
    characterData: true,
    childList: true,
    subtree: true,
  });

  return observer;
}

function translateTextNodes(root: ParentNode, locale: AppLocale) {
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
    acceptNode(node) {
      const parent = node.parentElement;
      if (!parent || shouldSkip(parent) || !node.nodeValue?.trim()) {
        return NodeFilter.FILTER_REJECT;
      }
      return NodeFilter.FILTER_ACCEPT;
    },
  });

  let node = walker.nextNode();
  while (node) {
    const current = node.nodeValue || '';
    const trimmed = current.trim();
    const translated = translateDomText(trimmed, locale);
    if (translated !== trimmed) {
      node.nodeValue = current.replace(trimmed, translated);
    }
    node = walker.nextNode();
  }
}

function translateAttributes(root: ParentNode, locale: AppLocale) {
  const elements =
    root instanceof Element
      ? [root, ...Array.from(root.querySelectorAll('*'))]
      : Array.from(root.querySelectorAll('*'));

  for (const element of elements) {
    if (shouldSkip(element)) {
      continue;
    }
    for (const attribute of translatableAttributes) {
      const value = element.getAttribute(attribute);
      if (!value?.trim()) {
        continue;
      }
      const translated = translateDomText(value.trim(), locale);
      if (translated !== value.trim()) {
        element.setAttribute(attribute, value.replace(value.trim(), translated));
      }
    }
  }
}

function shouldSkip(element: Element): boolean {
  return Boolean(element.closest(skippedSelector));
}

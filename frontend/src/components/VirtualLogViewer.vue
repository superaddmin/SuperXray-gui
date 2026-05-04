<template>
  <div ref="containerRef" class="log-viewport" @scroll="handleScroll">
    <div class="log-spacer" :style="{ height: `${totalHeight}px` }">
      <div class="log-window" :style="{ transform: `translateY(${offsetY}px)` }">
        <div
          v-for="item in visibleItems"
          :key="item.index"
          class="log-line"
          :style="{ height: `${rowHeight}px` }"
        >
          {{ item.line }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';

const props = withDefaults(
  defineProps<{
    autoFollow: boolean;
    lines: string[];
    rowHeight?: number;
  }>(),
  {
    rowHeight: 24,
  },
);

const emit = defineEmits<{
  'update:autoFollow': [value: boolean];
}>();

const containerRef = ref<HTMLDivElement | null>(null);
const scrollTop = ref(0);
const viewportHeight = ref(420);
const overscan = 8;

const totalHeight = computed(() => props.lines.length * props.rowHeight);
const startIndex = computed(() =>
  Math.max(0, Math.floor(scrollTop.value / props.rowHeight) - overscan),
);
const visibleCount = computed(
  () => Math.ceil(viewportHeight.value / props.rowHeight) + overscan * 2,
);
const endIndex = computed(() =>
  Math.min(props.lines.length, startIndex.value + visibleCount.value),
);
const offsetY = computed(() => startIndex.value * props.rowHeight);
const visibleItems = computed(() =>
  props.lines.slice(startIndex.value, endIndex.value).map((line, index) => ({
    index: startIndex.value + index,
    line,
  })),
);

watch(
  () => props.lines.length,
  () => {
    if (props.autoFollow) {
      void nextTick(scrollToBottom);
    }
  },
);

function handleScroll() {
  const container = containerRef.value;
  if (!container) {
    return;
  }

  scrollTop.value = container.scrollTop;
  viewportHeight.value = container.clientHeight;
  const distanceToBottom = container.scrollHeight - container.scrollTop - container.clientHeight;
  emit('update:autoFollow', distanceToBottom < props.rowHeight * 2);
}

function scrollToBottom() {
  const container = containerRef.value;
  if (!container) {
    return;
  }
  container.scrollTop = container.scrollHeight;
  scrollTop.value = container.scrollTop;
}
</script>

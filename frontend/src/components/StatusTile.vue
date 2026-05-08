<template>
  <ACard class="status-tile" :class="`status-tile-${tone}`" :bordered="false">
    <p class="tile-label">{{ label }}</p>
    <strong
      :class="{ 'status-value-compact': value.length > 18, 'status-value-long': value.length > 32 }"
    >
      {{ value }}
    </strong>
    <span>{{ hint }}</span>
  </ACard>
</template>

<script setup lang="ts">
import { Card as ACard } from 'ant-design-vue';

withDefaults(
  defineProps<{
    hint: string;
    label: string;
    tone?: 'danger' | 'info' | 'neutral' | 'success' | 'warning';
    value: string;
  }>(),
  { tone: 'info' },
);
</script>

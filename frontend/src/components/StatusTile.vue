<template>
  <ACard class="status-tile" :bordered="false">
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

defineProps<{
  hint: string;
  label: string;
  value: string;
}>();
</script>

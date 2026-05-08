<template>
  <header class="page-header">
    <div class="page-header-copy">
      <p v-if="eyebrow" class="page-eyebrow">{{ eyebrow }}</p>
      <h1>{{ title }}</h1>
      <p v-if="description" class="page-description">{{ description }}</p>
    </div>
    <div class="page-header-actions">
      <slot />
    </div>
  </header>
</template>

<script setup lang="ts">
defineProps<{
  description?: string;
  eyebrow?: string;
  title: string;
}>();
</script>

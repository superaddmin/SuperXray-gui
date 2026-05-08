<template>
  <section class="form-section">
    <div class="form-section-header">
      <div class="form-section-heading">
        <p v-if="eyebrow" class="page-eyebrow">{{ eyebrow }}</p>
        <h3 class="form-section-title">{{ title }}</h3>
        <p v-if="description" class="form-section-description">{{ description }}</p>
      </div>
      <div v-if="$slots.actions" class="form-actions">
        <slot name="actions" />
      </div>
    </div>
    <slot />
  </section>
</template>

<script setup lang="ts">
defineProps<{
  eyebrow?: string;
  title: string;
  description?: string;
}>();
</script>

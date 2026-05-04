<template>
  <main class="not-found">
    <AResult status="404" title="404" sub-title="The requested panel route was not found.">
      <template #extra>
        <RouterLink to="/">
          <AButton type="primary">Dashboard</AButton>
        </RouterLink>
      </template>
    </AResult>
  </main>
</template>

<script setup lang="ts">
import { Button as AButton, Result as AResult } from 'ant-design-vue';
import { RouterLink } from 'vue-router';
</script>

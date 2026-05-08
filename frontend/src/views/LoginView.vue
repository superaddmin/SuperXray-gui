<template>
  <main class="login-page">
    <AButton
      class="login-language-toggle"
      size="small"
      :aria-label="languageToggleAriaLabel"
      @click="appStore.toggleLocale"
    >
      {{ languageButtonLabel }}
    </AButton>
    <section class="login-card" aria-labelledby="login-title">
      <div class="login-mark" aria-hidden="true">
        <StarOutlined />
      </div>
      <h1 id="login-title">Control Xray with Confidence</h1>
      <p>Sign in to access the SuperXray operations console</p>

      <AForm
        class="login-form"
        layout="vertical"
        :model="formState"
        autocomplete="on"
        @finish="handleSubmit"
      >
        <AFormItem label="Username" name="username" :rules="usernameRules">
          <AInput
            v-model:value="formState.username"
            aria-label="Username"
            autocomplete="username"
            size="large"
            placeholder="admin"
          >
            <template #prefix>
              <UserOutlined />
            </template>
          </AInput>
        </AFormItem>

        <AFormItem label="Password" name="password" :rules="passwordRules">
          <AInputPassword
            v-model:value="formState.password"
            aria-label="Password"
            autocomplete="current-password"
            size="large"
            placeholder="Enter your password"
          >
            <template #prefix>
              <LockOutlined />
            </template>
          </AInputPassword>
        </AFormItem>

        <AFormItem
          v-if="twoFactorEnabled"
          label="Two-factor code"
          name="twoFactorCode"
          :rules="twoFactorRules"
        >
          <AInput
            v-model:value="formState.twoFactorCode"
            aria-label="Two-factor code"
            autocomplete="one-time-code"
            inputmode="numeric"
            size="large"
            placeholder="000000"
          >
            <template #prefix>
              <SafetyCertificateOutlined />
            </template>
          </AInput>
        </AFormItem>

        <AButton
          block
          class="login-submit"
          html-type="submit"
          :loading="submitting"
          size="large"
          type="primary"
        >
          Sign in
        </AButton>
      </AForm>
    </section>
  </main>
</template>

<script setup lang="ts">
import {
  LockOutlined,
  SafetyCertificateOutlined,
  StarOutlined,
  UserOutlined,
} from '@ant-design/icons-vue';
import {
  Button as AButton,
  Form as AForm,
  FormItem as AFormItem,
  Input as AInput,
  InputPassword as AInputPassword,
  message,
} from 'ant-design-vue';
import type { Rule } from 'ant-design-vue/es/form';
import { computed, onMounted, reactive, ref } from 'vue';

import { getTwoFactorEnabled, login } from '@/api/auth';
import { getLanguageButtonLabel, getLanguageToggleAriaLabel, translate } from '@/i18n/messages';
import { useAppStore } from '@/stores/app';
import { getRuntimeConfig } from '@/types/runtime';

const runtimeConfig = getRuntimeConfig();
const appStore = useAppStore();

const formState = reactive({
  password: '',
  twoFactorCode: '',
  username: '',
});

const submitting = ref(false);
const twoFactorEnabled = ref(false);

const languageButtonLabel = computed(() => getLanguageButtonLabel(appStore.locale));
const languageToggleAriaLabel = computed(() => getLanguageToggleAriaLabel(appStore.locale));
const usernameRules = computed<Rule[]>(() => [
  { required: true, message: translate('login.usernameRequired', appStore.locale) },
]);
const passwordRules = computed<Rule[]>(() => [
  { required: true, message: translate('login.passwordRequired', appStore.locale) },
]);
const twoFactorRules = computed<Rule[]>(() => [
  { required: true, message: translate('login.twoFactorRequired', appStore.locale) },
]);

onMounted(() => {
  void loadTwoFactorState();
});

async function loadTwoFactorState() {
  try {
    twoFactorEnabled.value = await getTwoFactorEnabled({
      notifyOnError: false,
      redirectOnUnauthorized: false,
    });
  } catch {
    twoFactorEnabled.value = false;
  }
}

async function handleSubmit() {
  submitting.value = true;
  try {
    await login(
      {
        password: formState.password,
        twoFactorCode: formState.twoFactorCode,
        username: formState.username,
      },
      { redirectOnUnauthorized: false },
    );
    window.location.assign(runtimeConfig.uiBasePath);
  } catch (error) {
    const text =
      error instanceof Error ? error.message : translate('login.signInFailed', appStore.locale);
    void message.error(text);
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <main class="login-page">
    <section class="login-card" aria-labelledby="login-title">
      <div class="login-mark" aria-hidden="true">
        <StarOutlined />
      </div>
      <h1 id="login-title">Welcome back</h1>
      <p>Sign in to access SuperXray</p>

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
import { onMounted, reactive, ref } from 'vue';

import { getTwoFactorEnabled, login } from '@/api/auth';
import { getRuntimeConfig } from '@/types/runtime';

const runtimeConfig = getRuntimeConfig();

const formState = reactive({
  password: '',
  twoFactorCode: '',
  username: '',
});

const submitting = ref(false);
const twoFactorEnabled = ref(false);

const usernameRules: Rule[] = [{ required: true, message: 'Username is required' }];
const passwordRules: Rule[] = [{ required: true, message: 'Password is required' }];
const twoFactorRules: Rule[] = [{ required: true, message: 'Two-factor code is required' }];

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
    const text = error instanceof Error ? error.message : 'Sign in failed';
    void message.error(text);
  } finally {
    submitting.value = false;
  }
}
</script>

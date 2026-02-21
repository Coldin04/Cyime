<script lang="ts">
  import { onMount } from 'svelte';

  type AuthProvider = {
    name: string;
    icon: string;
    ssoUrl: string;
  };

  let authProviders: AuthProvider[] = [];
  let isLoading = true;
  let error: string | null = null;

  onMount(async () => {
    try {
      const response = await fetch('/api/v1/auth/config');
      if (!response.ok) {
        throw new Error('无法获取认证配置');
      }
      const data = await response.json();
      authProviders = data.providers || [];
    } catch (e: any) {
      error = e.message;
      console.error('获取认证配置失败:', e);
    } finally {
      isLoading = false;
    }
  });
</script>

<div class="w-full rounded-2xl border-2 p-8 min-h-48">
  <h1 class="text-3xl font-semibold mb-4 py-2 text-gray-700 dark:text-gray-300">SSO 登录</h1>
  {#if isLoading}
    <div class="flex w-full h-full items-center justify-center rounded-xl">
      <p class="text-gray-500 dark:text-gray-400 py-4 h-24">正在加载登录选项...</p>
    </div>
  {:else if error}
    <div class="text-center text-red-500">
      <p>加载登录选项时发生错误:</p>
      <p class="font-mono text-sm">{error}</p>
    </div>
  {:else if authProviders.length > 0}
    <div class="flex flex-col space-y-4 py-2">
      {#each authProviders as provider, i}
        <a
          href={provider.ssoUrl}
          rel="external"
          class="block w-full rounded-xl py-3 px-6 text-center text-base font-medium shadow-sm transition-shadow hover:shadow-lg {i ===
          0
            ? 'bg-riptide-500 text-riptide-50'
            : 'bg-white text-gray-600 dark:bg-slate-700 dark:text-gray-300'}"
        >
          使用 {provider.name} 登录
        </a>
      {/each}
    </div>
  {:else}
    <div class="text-center text-gray-500 dark:text-gray-400">
      <p>当前没有可用的 SSO 登录选项。</p>
      <p class="mt-2 text-sm">请联系您的管理员进行配置。</p>
    </div>
  {/if}
</div>


<script lang="ts">
  import { onMount } from 'svelte';
  import * as m from '$paraglide/messages';

  type AuthProvider = {
    name: string;
    icon: string;
    ssoUrl: string;
  };

  let authProviders: AuthProvider[] = [];
  let isLoading = true;
  let error: string | null = null;

  function formatProviderName(name: string): string {
    const value = name.trim();
    if (!value) return name;
    return value.charAt(0).toUpperCase() + value.slice(1);
  }

  onMount(async () => {
    try {
      const response = await fetch('/api/v1/auth/config');
      if (!response.ok) {
        throw new Error(m.sso_login_box_error_fetch_config());
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
  <h1 class="text-3xl font-semibold mb-4 py-2 text-gray-700 dark:text-gray-300">{m.sso_login_box_title()}</h1>
  {#if isLoading}
    <div class="flex w-full h-full items-center justify-center rounded-xl">
      <p class="text-gray-500 dark:text-gray-400 py-4 h-24">{m.sso_login_box_loading_options()}</p>
    </div>
  {:else if error}
    <div class="text-center text-red-500">
      <p>{m.sso_login_box_error_loading_options()}</p>
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
          {m.sso_login_box_login_with_provider({ providerName: formatProviderName(provider.name) })}
        </a>
      {/each}
    </div>
  {:else}
    <div class="text-center text-gray-500 dark:text-gray-400">
      <p>{m.sso_login_box_no_sso_options()}</p>
      <p class="mt-2 text-sm">{m.sso_login_box_contact_admin_for_config()}</p>
    </div>
  {/if}
</div>

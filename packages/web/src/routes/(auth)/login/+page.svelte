<script lang="ts">
  import SsoLoginBox from '$lib/components/auth/SsoLoginBox.svelte';
  import Logo from '$lib/components/common/Logo.svelte';
  import * as m from '$paraglide/messages';
</script>

<svelte:head>
  <title>{m.page_title_login()}</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center bg-white py-12 dark:bg-slate-900 md:pb-0">
  <div class="container mx-auto p-4">
    <div class="mx-auto max-w-4xl grid grid-cols-1 items-center gap-6 md:grid-cols-2">
      <!-- Left Column: Brand and Tagline -->
      <div class="mb-6 flex flex-col justify-center text-center md:mb-0 md:-translate-y-7 md:pr-10 md:text-left">
        <Logo href="/" labelClass="text-5xl font-bold tracking-tight md:text-6xl" />
        <p class="mt-5 hidden text-sm leading-8 text-slate-600 dark:text-slate-400 md:inline-block md:text-lg">
          {m.login_page_tagline_part1()}<br />
          {m.login_page_tagline_part2()}
        </p>
      </div>

      <!-- Right Column: Login Box -->
      <div class="mx-auto w-full max-w-md md:-translate-y-4">
        <SsoLoginBox />
      </div>
    </div>
  </div>
</div>

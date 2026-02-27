<script lang="ts">
  import SsoLoginBox from '$lib/components/auth/SsoLoginBox.svelte';
  import * as m from '$paraglide/messages';
</script>

<div class="flex min-h-screen items-center justify-center bg-white py-12 dark:bg-slate-900 md:pb-0">
  <div class="container mx-auto p-4">
    <div class="mx-auto max-w-4xl grid grid-cols-1 items-center gap-6 md:grid-cols-2">
      <!-- Left Column: Brand and Tagline -->
      <div class="text-center md:text-left mb-6 md:mb-0">
        <h1 class="text-5xl font-bold md:text-6xl">
          <span class="bg-gradient-to-r from-cyan-300 to-yellow-300 bg-clip-text text-transparent">CyimeWrite</span>
        </h1>
        <p class="hidden md:inline-block mx-auto mt-4 text-sm md:text-lg text-gray-600 dark:text-gray-400">
          {m.login_page_tagline_part1()}<br>          {m.login_page_tagline_part2()}
        </p>
      </div>

      <!-- Right Column: Login Box -->
      <div class="w-full max-w-md mx-auto">
        <SsoLoginBox />
      </div>
    </div>
  </div>
</div>

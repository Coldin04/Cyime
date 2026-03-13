<script lang="ts">
	import * as m from '$paraglide/messages';
</script>

<svelte:head>
	<title>{m.page_title_user_center()}</title>
</svelte:head>

<section class="space-y-3">
	<h1 class="text-2xl font-bold text-zinc-900 dark:text-zinc-100">{m.user_center_title()}</h1>
	<p class="text-sm text-zinc-600 dark:text-zinc-400">
		{m.user_center_description()}
	</p>
	<div class="rounded-xl border border-zinc-200 bg-zinc-50 p-4 text-sm text-zinc-500 dark:border-zinc-700 dark:bg-zinc-800/50 dark:text-zinc-300">
		概览内容占位区
	</div>
</section>

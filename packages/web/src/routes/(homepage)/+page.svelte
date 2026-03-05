<script lang="ts">
  import * as m from '$paraglide/messages';
</script>

<svelte:head>
  <title>{m.page_title_homepage()}</title>
</svelte:head>

<div
	class="flex min-h-screen flex-col items-center justify-center bg-riptide-50 p-8 text-center dark:bg-slate-900"
>
	<h1 class="text-3xl font-bold sm:text-4xl md:text-6xl">
		<span class="bg-gradient-to-r from-cyan-300 to-yellow-300 bg-clip-text text-transparent">
			CyimeWrite
		</span>
		<span class="text-gray-800 dark:text-gray-200">{m.homepage_brand_name_chinese()}</span>
	</h1>
	<p class="mt-4 text-base text-gray-600 dark:text-gray-400 md:text-xl">
		{m.homepage_tagline_part1()}<br>		{m.homepage_tagline_part2()}
	</p>
	<div class="mt-8 flex flex-col space-y-4 sm:flex-row sm:space-y-0 sm:space-x-4">
		<a href="/workspace"
			class="rounded-xl bg-riptide-500 py-3 px-6 font-semibold shadow-lg transition-shadow hover:shadow-xl text-riptide-50"
		>
			{m.homepage_start_writing_button()}
		</a>
		<a
            href="#features"
			class="rounded-xl bg-white py-3 px-6 font-semibold text-gray-600 shadow-lg transition-shadow hover:shadow-xl dark:bg-slate-700 dark:text-gray-300"
		>
			{m.homepage_learn_more_button()}
		</a>
	</div>
</div>

<!-- Features Section -->
<div id="features" class="space-y-16 py-16 md:space-y-28 md:py-28">
	<!-- Feature 1: Online Sync -->
	<section class="bg-white px-8 dark:bg-slate-800">
		<div class="mx-auto max-w-5xl">
			<div class="flex flex-col items-center gap-8 md:flex-row md:gap-8">
				<div class="w-full md:w-1/2">
					<div
						class="aspect-video w-full rounded-2xl bg-gray-200 shadow-lg dark:bg-slate-700"
						aria-label={m.homepage_editor_features_screenshot_alt()}
					></div>
				</div>
				<div class="w-full text-center md:w-1/2 md:pl-16 md:text-left">
					<h2
						class="text-2xl font-bold text-gray-800 dark:text-gray-200 sm:text-3xl md:text-4xl"
					>
						{m.homepage_online_sync_title()}
					</h2>
					<p class="mt-4 text-sm text-gray-600 dark:text-gray-400 md:text-lg font-light">
						{m.homepage_online_sync_description()}
					</p>
				</div>
			</div>
		</div>
	</section>

	<!-- Feature 2: Minimalist Interface -->
	<section class="bg-riptide-50 py-16 dark:bg-slate-900 md:py-28">
		<div class="mx-auto max-w-5xl px-8">
			<div class="flex flex-col items-center gap-8 md:flex-row-reverse md:gap-8">
				<div class="w-full md:w-1/2">
					<div
						class="aspect-video w-full rounded-2xl bg-gray-200 shadow-lg dark:bg-slate-700"
						aria-label={m.homepage_minimalist_editor_screenshot_alt()}
					></div>
				</div>
				<div class="w-full text-center md:w-1/2 md:pr-16 md:text-left">
					<h2
						class="text-2xl font-bold text-gray-800 dark:text-gray-200 sm:text-3xl md:text-4xl"
					>
						{m.homepage_focus_writing_title()}
					</h2>
					<p class="mt-4 text-sm text-gray-600 dark:text-gray-400 md:text-lg font-light">
						{m.homepage_focus_writing_description()}
					</p>
				</div>
			</div>
		</div>
	</section>

	<!-- Feature 3: Smooth Response -->
	<section class="bg-white px-8 dark:bg-slate-800">
		<div class="mx-auto max-w-5xl">
			<div class="flex flex-col items-center gap-8 md:flex-row md:gap-8">
				<div class="w-full md:w-1/2">
					<div
						class="aspect-video w-full rounded-2xl bg-gray-200 shadow-lg dark:bg-slate-700"
						aria-label={m.homepage_smooth_input_animation_alt()}
					></div>
				</div>
				<div class="w-full text-center md:w-1/2 md:pl-16 md:text-left">
					<h2
						class="text-2xl font-bold text-gray-800 dark:text-gray-200 sm:text-3xl md:text-4xl"
					>
						{m.homepage_smooth_editor_title()}
					</h2>
					<p class="mt-4 text-sm text-gray-600 dark:text-gray-400 md:text-lg font-light">
						{m.homepage_smooth_editor_description()}
					</p>
				</div>
			</div>
		</div>
	</section>
</div>

<!-- Footer -->
<footer class="bg-gray-100 dark:bg-slate-700">
	<div class="mx-auto max-w-7xl py-12 px-4 text-center sm:px-6 lg:px-8">
		<p class="text-gray-500 dark:text-gray-300">{m.homepage_footer_copyright()}</p>
	</div>
</footer>

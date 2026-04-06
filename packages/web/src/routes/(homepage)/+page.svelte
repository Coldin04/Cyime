<script lang="ts">
  import * as m from '$paraglide/messages';
</script>

<svelte:head>
  <title>{m.page_title_homepage()}</title>
  <meta name="description" content={m.homepage_meta_description()} />
  <meta name="keywords" content={m.homepage_meta_keywords()} />
  <meta property="og:title" content={m.page_title_homepage()} />
  <meta property="og:description" content={m.homepage_meta_description()} />
  <meta name="twitter:title" content={m.page_title_homepage()} />
  <meta name="twitter:description" content={m.homepage_meta_description()} />
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

<section class="bg-white px-6 py-14 dark:bg-slate-800 sm:px-8">
	<div class="mx-auto max-w-5xl">
		<div class="rounded-3xl border border-riptide-100 bg-riptide-50/70 p-6 shadow-sm dark:border-slate-700 dark:bg-slate-900/70 sm:p-8">
			<p class="text-xs font-semibold uppercase tracking-[0.24em] text-riptide-600 dark:text-riptide-300">
				{m.homepage_intro_eyebrow()}
			</p>
			<h2 class="mt-3 text-2xl font-bold text-gray-900 dark:text-gray-100 sm:text-3xl">
				{m.homepage_intro_title()}
			</h2>
			<p class="mt-4 max-w-3xl text-sm leading-7 text-gray-600 dark:text-gray-300 sm:text-base">
				{m.homepage_intro_description()}
			</p>
			<div class="mt-6 flex flex-wrap gap-3">
				<span class="rounded-full bg-white px-4 py-2 text-sm font-medium text-gray-700 shadow-sm dark:bg-slate-800 dark:text-gray-200">{m.homepage_capability_sync()}</span>
				<span class="rounded-full bg-white px-4 py-2 text-sm font-medium text-gray-700 shadow-sm dark:bg-slate-800 dark:text-gray-200">{m.homepage_capability_collab()}</span>
				<span class="rounded-full bg-white px-4 py-2 text-sm font-medium text-gray-700 shadow-sm dark:bg-slate-800 dark:text-gray-200">{m.homepage_capability_share()}</span>
				<span class="rounded-full bg-white px-4 py-2 text-sm font-medium text-gray-700 shadow-sm dark:bg-slate-800 dark:text-gray-200">{m.homepage_capability_assets()}</span>
			</div>
			<div class="mt-8 grid gap-4 md:grid-cols-2">
				<div class="rounded-2xl bg-white px-5 py-4 shadow-sm dark:bg-slate-800">
					<p class="text-xs font-medium uppercase tracking-wide text-gray-400">{m.homepage_attribute_product_label()}</p>
					<p class="mt-2 text-sm font-medium text-gray-800 dark:text-gray-100">{m.homepage_attribute_product_value()}</p>
				</div>
				<div class="rounded-2xl bg-white px-5 py-4 shadow-sm dark:bg-slate-800">
					<p class="text-xs font-medium uppercase tracking-wide text-gray-400">{m.homepage_attribute_editor_label()}</p>
					<p class="mt-2 text-sm font-medium text-gray-800 dark:text-gray-100">{m.homepage_attribute_editor_value()}</p>
				</div>
				<div class="rounded-2xl bg-white px-5 py-4 shadow-sm dark:bg-slate-800">
					<p class="text-xs font-medium uppercase tracking-wide text-gray-400">{m.homepage_attribute_collab_label()}</p>
					<p class="mt-2 text-sm font-medium text-gray-800 dark:text-gray-100">{m.homepage_attribute_collab_value()}</p>
				</div>
				<div class="rounded-2xl bg-white px-5 py-4 shadow-sm dark:bg-slate-800">
					<p class="text-xs font-medium uppercase tracking-wide text-gray-400">{m.homepage_attribute_sharing_label()}</p>
					<p class="mt-2 text-sm font-medium text-gray-800 dark:text-gray-100">{m.homepage_attribute_sharing_value()}</p>
				</div>
			</div>
		</div>
	</div>
</section>

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

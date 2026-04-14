<script lang="ts">
	import * as m from '$paraglide/messages';
	
	import { onMount } from 'svelte';

	const homepageHeroHeadlinePhrases = [
		m.homepage_hero_word_light(),
		m.homepage_hero_word_flow()
	];

	let homepageHeroHeadlinePhrase = homepageHeroHeadlinePhrases[0];
	let homepageHeroHeadlinePhraseIndex = 0;
	
	onMount(() => {
		let rotationTimeout: ReturnType<typeof setTimeout>;

		const scheduleNextPhraseRotation = () => {
			rotationTimeout = setTimeout(() => {
				homepageHeroHeadlinePhraseIndex =
					(homepageHeroHeadlinePhraseIndex + 1) % homepageHeroHeadlinePhrases.length;
				homepageHeroHeadlinePhrase =
					homepageHeroHeadlinePhrases[homepageHeroHeadlinePhraseIndex];
				scheduleNextPhraseRotation();
			}, 3000);
		};

		scheduleNextPhraseRotation();

		return () => clearTimeout(rotationTimeout);
	});
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
	class="homepage-hero min-h-screen px-8 py-10 dark:bg-slate-900"
>
	<div class="mx-auto flex min-h-[calc(100vh-5rem)] max-w-6xl flex-col">

		<div class="flex flex-1 flex-col items-center justify-center text-center">
			<h1
				class="max-w-5xl text-4xl font-bold leading-[1.14] tracking-tight text-slate-800 dark:text-slate-100 sm:text-5xl md:leading-[1.1] md:text-7xl"
			>
				{#key `${homepageHeroHeadlinePhraseIndex}-${homepageHeroHeadlinePhrase}`}
					<span
						class="homepage-hero-headline-phrase slide-in bg-gradient-to-r from-teal-400 to-sky-300 bg-clip-text text-transparent"
					>
						{homepageHeroHeadlinePhrase}
					</span>
				{/key}
				<span class="mt-3 block md:mt-4">{m.homepage_hero_suffix()}</span>
			</h1>
			<p class="mt-8 max-w-3xl text-base leading-8 text-slate-500 dark:text-slate-400 md:text-2xl">
				{m.homepage_hero_description()}
			</p>
				<div class="mt-8 flex flex-col space-y-4 sm:flex-row sm:space-y-0 sm:space-x-4">
			<a href="/workspace"
				class="rounded-xl bg-sky-500 py-3 px-6 font-semibold text-white shadow-lg transition-shadow "
			>
				{m.homepage_start_writing_button()}
			</a>
		<a
            href="#features"
			class="rounded-xl bg-sky-50 py-3 px-6 font-semibold text-slate-800 shadow-lg transition-shadow hover:shadow-xl dark:bg-slate-700 dark:text-gray-300"
		>
			{m.homepage_learn_more_button()}
		</a>
				</div>
		</div>
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
	<section class="bg-sky-50 py-16 dark:bg-slate-900 md:py-28">
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


<style>
	@keyframes slideInFromRight {
		0% {
			transform: translateX(1.5rem);
			clip-path: inset(0 0 0 100%);
			opacity: 0;
		}
		45% {
			opacity: 1;
		}
		100% {
			transform: translateX(0);
			clip-path: inset(0 0 0 0);
			opacity: 1;
		}
	}

	.homepage-hero-headline-phrase {
		display: inline-block;
		will-change: transform, clip-path, opacity;
	}
	
	.slide-in {
		animation: slideInFromRight 0.85s cubic-bezier(0.22, 1, 0.36, 1);
	}
</style>

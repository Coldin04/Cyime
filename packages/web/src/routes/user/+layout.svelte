<script lang="ts">
	import * as m from '$paraglide/messages';
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import TopBar from '$lib/components/workspace/TopBar.svelte';
	import UserAvatar from '$lib/components/common/UserAvatar.svelte';
	import { auth } from '$lib/stores/auth';
	import House from '~icons/ph/house';
	import UserCircle from '~icons/ph/user-circle';
	import ShieldCheck from '~icons/ph/shield-check';
	import ImagesSquare from '~icons/ph/images-square';
	import LinkSimple from '~icons/ph/link-simple';
	import CaretDown from '~icons/ph/caret-down';

	let { children } = $props();
	let mobileNavOpen = $state(false);

	const navItems = [
		{ href: '/user', label: m.user_nav_overview(), icon: House },
		{ href: '/user/profile', label: m.user_nav_profile(), icon: UserCircle },
		{ href: '/user/security', label: m.user_nav_security(), icon: ShieldCheck },
		{ href: '/user/image-beds', label: m.user_nav_image_beds(), icon: LinkSimple },
		{ href: '/user/media', label: m.user_nav_media_library(), icon: ImagesSquare }
	];

	function isActive(pathname: string, href: string): boolean {
		if (href === '/user') return pathname === href;
		return pathname.startsWith(href);
	}

	$effect(() => {
		$page.url.pathname;
		mobileNavOpen = false;
	});

	$effect(() => {
		if (browser && !$auth.loading && !$auth.token) {
			goto('/login', { replaceState: true });
		}
	});
</script>

{#if $auth.loading}
	<div class="flex h-screen w-full items-center justify-center bg-gray-50 dark:bg-gray-900">
		<p class="text-lg text-gray-600 dark:text-gray-300">{m.workspace_loading()}</p>
	</div>
{:else if $auth.token}
	<TopBar />
	<main class="mx-auto grid max-w-6xl grid-cols-1 gap-6 px-4 py-8 sm:px-6 lg:grid-cols-[240px_minmax(0,1fr)]">
		<aside class="rounded-xl border border-zinc-200 bg-white p-2 dark:border-zinc-700 dark:bg-zinc-900 lg:hidden">
			<button
				type="button"
				class="flex w-full items-center justify-between rounded-lg px-3 py-2 text-left"
				onclick={() => (mobileNavOpen = !mobileNavOpen)}
			>
				<div class="flex min-w-0 items-center gap-3">
					<UserAvatar size={40} name={$auth.user?.displayName} avatarUrl={$auth.user?.avatarUrl} />
					<div class="min-w-0">
						<p class="truncate text-sm font-semibold text-zinc-900 dark:text-zinc-100">
							{$auth.user?.displayName || m.user_common_default_name()}
						</p>
						<p class="truncate text-xs text-zinc-500 dark:text-zinc-400">{$auth.user?.email || m.user_common_no_email()}</p>
					</div>
				</div>
				<CaretDown class={`h-4 w-4 text-zinc-500 transition-transform ${mobileNavOpen ? 'rotate-180' : ''}`} />
			</button>
			{#if mobileNavOpen}
				<nav class="space-y-1 px-1 pb-1">
					{#each navItems as item (item.href)}
						<a
							href={item.href}
							class={`flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition-colors ${
								isActive($page.url.pathname, item.href)
									? 'bg-riptide-100 font-semibold text-riptide-800 dark:bg-riptide-900/50 dark:text-riptide-200'
									: 'text-zinc-600 hover:bg-zinc-100 hover:text-zinc-900 dark:text-zinc-300 dark:hover:bg-zinc-800 dark:hover:text-zinc-100'
							}`}
						>
							<item.icon class="h-4 w-4 flex-shrink-0" />
							{item.label}
						</a>
					{/each}
				</nav>
			{/if}
		</aside>
		<aside class="hidden space-y-5 rounded-2xl border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900 lg:block">
			<div class="flex items-center gap-3">
				<UserAvatar size={44} name={$auth.user?.displayName} avatarUrl={$auth.user?.avatarUrl} />
				<div class="min-w-0">
					<p class="truncate text-sm font-semibold text-zinc-900 dark:text-zinc-100">
						{$auth.user?.displayName || m.user_common_default_name()}
					</p>
					<p class="truncate text-xs text-zinc-500 dark:text-zinc-400">{$auth.user?.email || m.user_common_no_email()}</p>
				</div>
			</div>
			<nav class="space-y-1">
				{#each navItems as item (item.href)}
					<a
						href={item.href}
						class={`flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition-colors ${
							isActive($page.url.pathname, item.href)
								? 'bg-riptide-100 font-semibold text-riptide-800 dark:bg-riptide-900/50 dark:text-riptide-200'
								: 'text-zinc-600 hover:bg-zinc-100 hover:text-zinc-900 dark:text-zinc-300 dark:hover:bg-zinc-800 dark:hover:text-zinc-100'
						}`}
					>
						<item.icon class="h-4 w-4 flex-shrink-0" />
						{item.label}
					</a>
				{/each}
			</nav>
		</aside>
		<section class="min-w-0 rounded-2xl border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900 sm:p-6">
			{@render children()}
		</section>
	</main>
{/if}

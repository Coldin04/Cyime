<script lang="ts">
	import type { OutgoingSharedDocumentItem } from '$lib/api/workspace';
	import FileText from '~icons/ph/file-text';
	import Table from '~icons/ph/table';
	import UsersThree from '~icons/ph/users-three';
	import GlobeHemisphereWest from '~icons/ph/globe-hemisphere-west';
	import Lock from '~icons/ph/lock';

	let {
		doc,
		onManageMembers,
		onManagePublicAccess
	}: {
		doc: OutgoingSharedDocumentItem;
		onManageMembers: () => void;
		onManagePublicAccess: () => void;
	} = $props();

	function publicAccessLabel(access: string) {
		switch (access) {
			case 'public':
				return '公开';
			case 'authenticated':
				return '登录';
			default:
				return '私有';
		}
	}
</script>

<div class="flex items-center justify-between gap-4 border-b border-zinc-200 px-4 py-3 dark:border-zinc-700">
	<div class="flex min-w-0 items-start gap-3">
		{#if doc.documentType === 'table'}
			<Table class="mt-0.5 h-5 w-5 shrink-0 text-blue-500 dark:text-blue-400" />
		{:else}
			<FileText class="mt-0.5 h-5 w-5 shrink-0 text-blue-500 dark:text-blue-400" />
		{/if}

		<div class="min-w-0">
			<p class="truncate text-sm font-medium text-zinc-800 dark:text-zinc-200">{doc.title}</p>
			{#if doc.excerpt}
				<p class="mt-0.5 line-clamp-1 text-xs text-zinc-500 dark:text-zinc-400">{doc.excerpt}</p>
			{/if}
		</div>
	</div>

	<div class="flex shrink-0 items-center gap-4 text-xs text-zinc-500 dark:text-zinc-400">
		<button
			type="button"
			class="inline-flex items-center gap-1.5 rounded-md px-1.5 py-1 transition hover:text-zinc-900 dark:hover:text-zinc-100"
			onclick={onManageMembers}
			title="管理成员"
		>
			<UsersThree class="h-4 w-4" />
			<span>{doc.sharedMemberCount} 人</span>
		</button>
		<button
			type="button"
			class="inline-flex items-center gap-1.5 rounded-md px-1.5 py-1 transition hover:text-zinc-900 dark:hover:text-zinc-100"
			onclick={onManagePublicAccess}
			title="公开访问"
		>
			{#if doc.publicAccess === 'private'}
				<Lock class="h-4 w-4" />
			{:else}
				<GlobeHemisphereWest class="h-4 w-4" />
			{/if}
			<span>{publicAccessLabel(doc.publicAccess)}</span>
		</button>
	</div>
</div>

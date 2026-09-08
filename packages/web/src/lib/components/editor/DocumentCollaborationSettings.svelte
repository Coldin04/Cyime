<script lang="ts">
	import {
		inviteDocumentByEmail,
		listDocumentMembers,
		removeDocumentMember,
		transferDocumentOwnership,
		type ShareDocumentMember
	} from '$lib/api/workspace';
	import { portal } from '$lib/actions/portal';
	import { toast } from 'svelte-sonner';
	import Minus from '~icons/ph/minus';
	import ArrowsLeftRight from '~icons/ph/arrows-left-right';
	import Plus from '~icons/ph/plus';
	import UsersThree from '~icons/ph/users-three';
	import X from '~icons/ph/x';
	import { goto } from '$app/navigation';

	type Props = {
		documentId: string;
		enabled?: boolean;
	};

	let { documentId, enabled = false }: Props = $props();

	let members = $state<ShareDocumentMember[]>([]);
	let isLoading = $state(false);
	let loadError = $state('');
	let showInviteDialog = $state(false);
	let isSubmittingInvite = $state(false);
	let removingUserId = $state<string | null>(null);
	let transferringUserId = $state<string | null>(null);
	let inviteEmail = $state('');

	$effect(() => {
		if (!enabled) return;
		void refreshMembers();
	});

	async function refreshMembers() {
		isLoading = true;
		loadError = '';
		try {
			const response = await listDocumentMembers(documentId);
			members = response.members;
		} catch (error) {
			loadError = error instanceof Error ? error.message : '加载成员失败';
		} finally {
			isLoading = false;
		}
	}

	function roleLabel(role: string) {
		return role === 'owner' ? '所有者' : '查看者';
	}

	function roleClass(role: string) {
		switch (role) {
			case 'owner':
				return 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900';
			default:
				return 'bg-zinc-200 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300';
		}
	}

	async function submitInvite() {
		const email = inviteEmail.trim();
		if (!email || isSubmittingInvite) {
			return;
		}

		isSubmittingInvite = true;
		try {
			const response = await inviteDocumentByEmail(documentId, email, 'viewer');
			members = response.members;
			showInviteDialog = false;
			inviteEmail = '';
			toast.success('邀请已发送');
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '发送邀请失败');
		} finally {
			isSubmittingInvite = false;
		}
	}

	async function removeMember(member: ShareDocumentMember) {
		if (member.role === 'owner' || removingUserId) {
			return;
		}
		const displayName = member.displayName || member.userId;
		const ok = window.confirm(`确认移除成员「${displayName}」吗？`);
		if (!ok) return;

		removingUserId = member.userId;
		try {
			const response = await removeDocumentMember(documentId, member.userId);
			members = response.members;
			toast.success('成员已移除');
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '移除成员失败');
		} finally {
			removingUserId = null;
		}
	}

	async function transferToMember(member: ShareDocumentMember) {
		if (member.role === 'owner' || transferringUserId) return;
		const displayName = member.displayName || member.email || member.userId;
		const confirmed = window.confirm(
			`确认将文档所有权转移给「${displayName}」吗？转移后你将变为查看者。`
		);
		if (!confirmed) return;

		transferringUserId = member.userId;
		try {
			await transferDocumentOwnership(documentId, member.userId);
			toast.success('文档所有权已转移');
			await goto(`/view/documents/${documentId}`);
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '转移文档失败');
		} finally {
			transferringUserId = null;
		}
	}

</script>

<div class="rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-950">
	<div class="flex items-center justify-between border-b border-zinc-200 px-4 py-3 dark:border-zinc-800">
		<div class="flex items-center gap-2 text-sm font-medium text-zinc-800 dark:text-zinc-100">
			<UsersThree class="h-4 w-4" />
			<span>成员权限</span>
		</div>
		<button
			type="button"
			class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-zinc-200 text-zinc-600 transition hover:bg-zinc-100 dark:border-zinc-700 dark:text-zinc-300 dark:hover:bg-zinc-800"
			onclick={() => (showInviteDialog = true)}
			title="邀请成员"
			aria-label="邀请成员"
		>
			<Plus class="h-4 w-4" />
		</button>
	</div>

	<div>
		{#if isLoading}
			<div class="p-4">
			<p class="text-sm text-zinc-500 dark:text-zinc-400">正在加载成员...</p>
			</div>
		{:else if loadError}
			<div class="p-4">
				<div class="flex items-center justify-between gap-3">
					<p class="text-sm text-rose-600 dark:text-rose-300">{loadError}</p>
					<button
						type="button"
						class="rounded-md border border-zinc-200 px-2 py-1 text-xs text-zinc-600 dark:border-zinc-700 dark:text-zinc-300"
						onclick={() => void refreshMembers()}
					>
						重试
					</button>
				</div>
			</div>
		{:else if members.length === 0}
			<div class="p-4">
			<p class="text-sm text-zinc-500 dark:text-zinc-400">当前还没有查看者</p>
			</div>
		{:else}
			<ul class="divide-y divide-zinc-200 dark:divide-zinc-800">
				{#each members as member (member.userId)}
					<li class="flex items-center justify-between px-4 py-3">
						<div class="min-w-0">
							<p class="truncate text-sm font-medium text-zinc-800 dark:text-zinc-100">
								{member.displayName || member.userId}
							</p>
							<p class="text-xs text-zinc-500 dark:text-zinc-400">{member.email || member.userId}</p>
						</div>
						<div class="ml-3 flex items-center gap-2">
							<span class={`rounded-full px-2 py-1 text-xs ${roleClass(member.role)}`}>
								{roleLabel(member.role)}
							</span>
							{#if member.role !== 'owner'}
								<button
									type="button"
									class="inline-flex h-6 w-6 items-center justify-center rounded-full border border-zinc-200 text-zinc-500 transition hover:border-sky-300 hover:text-sky-600 disabled:cursor-not-allowed disabled:opacity-50 dark:border-zinc-700 dark:text-zinc-400 dark:hover:border-sky-600/50 dark:hover:text-sky-300"
									title="转移所有权"
									aria-label="转移所有权"
									onclick={() => void transferToMember(member)}
									disabled={transferringUserId !== null || removingUserId !== null}
								>
									<ArrowsLeftRight class="h-3.5 w-3.5" />
								</button>
								<button
									type="button"
									class="inline-flex h-6 w-6 items-center justify-center rounded-full border border-zinc-200 text-zinc-500 transition hover:border-rose-300 hover:text-rose-600 disabled:cursor-not-allowed disabled:opacity-50 dark:border-zinc-700 dark:text-zinc-400 dark:hover:border-rose-600/50 dark:hover:text-rose-300"
									title="移除成员"
									aria-label="移除成员"
									onclick={() => void removeMember(member)}
									disabled={removingUserId !== null}
								>
									<Minus class="h-3.5 w-3.5" />
								</button>
							{/if}
						</div>
					</li>
				{/each}
			</ul>
		{/if}
	</div>
</div>

{#if showInviteDialog}
	<div
		use:portal
		class="fixed inset-0 z-[140] flex items-center justify-center bg-black/40 p-4"
		role="presentation"
		onclick={() => (showInviteDialog = false)}
	>
		<div
			class="w-full max-w-md rounded-xl border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-950"
			role="dialog"
			aria-modal="true"
			aria-label="邀请查看者"
			tabindex="-1"
			onclick={(event) => event.stopPropagation()}
			onkeydown={(event) => {
				if (event.key === 'Escape') {
					showInviteDialog = false;
				}
			}}
		>
			<div class="mb-4 flex items-center justify-between">
				<h4 class="text-sm font-semibold text-zinc-900 dark:text-zinc-100">邀请查看者</h4>
				<button
					type="button"
					class="rounded-md p-1 text-zinc-500 hover:bg-zinc-100 dark:hover:bg-zinc-800"
					onclick={() => (showInviteDialog = false)}
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			<div class="space-y-3">
				<div class="space-y-1">
					<label for="invite-email" class="text-xs text-zinc-500 dark:text-zinc-400">邮箱</label>
					<input
						id="invite-email"
						type="email"
						bind:value={inviteEmail}
						placeholder="name@example.com"
						class="w-full rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-900 outline-none focus:border-zinc-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100"
					/>
				</div>

				<p class="text-xs leading-5 text-zinc-500 dark:text-zinc-400">
					受邀者只能查看文档，不能修改正文、设置或成员权限。
				</p>
			</div>

			<div class="mt-4 flex justify-end gap-2">
				<button
					type="button"
					class="rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-700 dark:border-zinc-700 dark:text-zinc-300"
					onclick={() => (showInviteDialog = false)}
				>
					取消
				</button>
				<button
					type="button"
					class="rounded-md bg-sky-500 px-3 py-2 text-sm text-white shadow-sm disabled:opacity-60 dark:bg-sky-500 dark:text-white"
					onclick={() => void submitInvite()}
					disabled={isSubmittingInvite}
				>
					{isSubmittingInvite ? '发送中...' : '发送邀请'}
				</button>
			</div>
		</div>
	</div>
{/if}

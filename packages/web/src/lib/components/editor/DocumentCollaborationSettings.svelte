<script lang="ts">
	import {
		inviteDocumentByEmail,
		listDocumentMembers,
		removeDocumentMember,
		updateDocumentMemberRole,
		type ShareDocumentMember
	} from '$lib/api/workspace';
	import { portal } from '$lib/actions/portal';
	import { toast } from 'svelte-sonner';
	import Minus from '~icons/ph/minus';
	import Plus from '~icons/ph/plus';
	import UsersThree from '~icons/ph/users-three';
	import X from '~icons/ph/x';

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
	let roleEditingMember = $state<ShareDocumentMember | null>(null);
	let editingRole = $state<'viewer' | 'editor' | 'collaborator'>('editor');
	let isSubmittingRole = $state(false);
	let inviteEmail = $state('');
	let inviteRole = $state<'viewer' | 'editor' | 'collaborator'>('editor');

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
		switch (role) {
			case 'owner':
				return '所有者';
			case 'collaborator':
				return '协同者';
			case 'editor':
				return '编辑者';
			default:
				return '查看者';
		}
	}

	function roleClass(role: string) {
		switch (role) {
			case 'owner':
				return 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900';
			case 'collaborator':
				return 'bg-sky-100 text-sky-700 dark:bg-sky-900/40 dark:text-sky-300';
			case 'editor':
				return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300';
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
			const response = await inviteDocumentByEmail(documentId, email, inviteRole);
			members = response.members;
			showInviteDialog = false;
			inviteEmail = '';
			inviteRole = 'editor';
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

	function openRoleDialog(member: ShareDocumentMember) {
		if (member.role === 'owner') {
			return;
		}
		roleEditingMember = member;
		if (member.role === 'viewer' || member.role === 'editor' || member.role === 'collaborator') {
			editingRole = member.role;
		} else {
			editingRole = 'editor';
		}
	}

	async function submitRoleUpdate() {
		if (!roleEditingMember || isSubmittingRole) {
			return;
		}
		isSubmittingRole = true;
		try {
			const response = await updateDocumentMemberRole(documentId, roleEditingMember.userId, editingRole);
			members = response.members;
			toast.success('成员角色已更新');
			roleEditingMember = null;
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '更新角色失败');
		} finally {
			isSubmittingRole = false;
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
			<p class="text-sm text-zinc-500 dark:text-zinc-400">当前还没有协作者</p>
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
							<button
								type="button"
								class={`rounded-full px-2 py-1 text-xs transition ${roleClass(member.role)} ${
									member.role === 'owner' ? 'cursor-default' : 'hover:opacity-80'
								}`}
								onclick={() => openRoleDialog(member)}
								disabled={member.role === 'owner'}
							>
								{roleLabel(member.role)}
							</button>
							{#if member.role !== 'owner'}
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
			aria-label="邀请协作者"
			tabindex="-1"
			onclick={(event) => event.stopPropagation()}
			onkeydown={(event) => {
				if (event.key === 'Escape') {
					showInviteDialog = false;
				}
			}}
		>
			<div class="mb-4 flex items-center justify-between">
				<h4 class="text-sm font-semibold text-zinc-900 dark:text-zinc-100">邀请协作者</h4>
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

				<div class="space-y-1">
					<label for="invite-role" class="text-xs text-zinc-500 dark:text-zinc-400">权限</label>
					<select
						id="invite-role"
						bind:value={inviteRole}
						class="w-full rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-900 outline-none focus:border-zinc-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100"
					>
						<option value="viewer">查看者</option>
						<option value="editor">编辑者</option>
						<option value="collaborator">协同者</option>
					</select>
				</div>
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

{#if roleEditingMember}
	<div
		use:portal
		class="fixed inset-0 z-[145] flex items-center justify-center bg-black/45 p-4"
		role="presentation"
		onclick={() => (roleEditingMember = null)}
	>
		<div
			class="w-full max-w-sm rounded-xl border border-zinc-200 bg-white p-4 shadow-xl dark:border-zinc-800 dark:bg-zinc-950"
			role="dialog"
			aria-modal="true"
			aria-label="修改成员角色"
			tabindex="-1"
			onclick={(event) => event.stopPropagation()}
			onkeydown={(event) => {
				if (event.key === 'Escape') {
					roleEditingMember = null;
				}
			}}
		>
			<div class="mb-3 flex items-center justify-between">
				<h4 class="text-sm font-semibold text-zinc-900 dark:text-zinc-100">修改成员角色</h4>
				<button
					type="button"
					class="rounded-md p-1 text-zinc-500 hover:bg-zinc-100 dark:hover:bg-zinc-800"
					onclick={() => (roleEditingMember = null)}
				>
					<X class="h-4 w-4" />
				</button>
			</div>
			<p class="mb-3 text-xs text-zinc-500 dark:text-zinc-400">
				{roleEditingMember.displayName || roleEditingMember.userId} · {roleEditingMember.email || roleEditingMember.userId}
			</p>
			<select
				bind:value={editingRole}
				class="w-full rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-900 outline-none focus:border-zinc-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100"
			>
				<option value="viewer">查看者</option>
				<option value="editor">编辑者</option>
				<option value="collaborator">协同者</option>
			</select>
			<div class="mt-4 flex justify-end gap-2">
				<button
					type="button"
					class="rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-700 dark:border-zinc-700 dark:text-zinc-300"
					onclick={() => (roleEditingMember = null)}
				>
					取消
				</button>
				<button
					type="button"
					class="rounded-md bg-sky-500 px-3 py-2 text-sm text-white shadow-sm disabled:opacity-60 dark:bg-sky-500 dark:text-white"
					onclick={() => void submitRoleUpdate()}
					disabled={isSubmittingRole}
				>
					{isSubmittingRole ? '保存中...' : '保存'}
				</button>
			</div>
		</div>
	</div>
{/if}

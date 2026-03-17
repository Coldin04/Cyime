<script lang="ts">
	import * as m from '$paraglide/messages';
	import TableIcon from '~icons/ph/table';

	interface Props {
		isTableActive: boolean;
		canInsertTable: boolean;
		canAddRow: boolean;
		canAddColumn: boolean;
		canDeleteTable: boolean;
		onInsertTable: () => void;
		onAddRow: () => void;
		onAddColumn: () => void;
		onDeleteTable: () => void;
	}

	let {
		isTableActive,
		canInsertTable,
		canAddRow,
		canAddColumn,
		canDeleteTable,
		onInsertTable,
		onAddRow,
		onAddColumn,
		onDeleteTable
	}: Props = $props();

	const activeToggleClass = 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900';
	const inactiveToggleClass =
		'text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800';
	const pillButtonBaseClass =
		'inline-flex h-8 shrink-0 items-center gap-1 rounded-md px-2 text-xs leading-none transition-colors disabled:cursor-not-allowed disabled:opacity-50';
</script>

<button
	type="button"
	title={m.editor_toolbar_insert_table()}
	aria-label={m.editor_toolbar_insert_table()}
	disabled={!canInsertTable}
	class={`${pillButtonBaseClass} ${isTableActive ? activeToggleClass : inactiveToggleClass}`}
	onclick={onInsertTable}
>
	<TableIcon class="h-4 w-4" />
</button>

{#if isTableActive}
	<button
		type="button"
		title={m.editor_toolbar_add_row()}
		aria-label={m.editor_toolbar_add_row()}
		disabled={!canAddRow}
		class={`${pillButtonBaseClass} ${inactiveToggleClass}`}
		onclick={onAddRow}
	>
		<span>+R</span>
	</button>
	<button
		type="button"
		title={m.editor_toolbar_add_column()}
		aria-label={m.editor_toolbar_add_column()}
		disabled={!canAddColumn}
		class={`${pillButtonBaseClass} ${inactiveToggleClass}`}
		onclick={onAddColumn}
	>
		<span>+C</span>
	</button>
	<button
		type="button"
		title={m.editor_toolbar_delete_table()}
		aria-label={m.editor_toolbar_delete_table()}
		disabled={!canDeleteTable}
		class={`${pillButtonBaseClass} text-red-600 hover:bg-red-50 dark:text-red-300 dark:hover:bg-red-950/30`}
		onclick={onDeleteTable}
	>
		<span>-T</span>
	</button>
{/if}

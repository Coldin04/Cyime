<script lang="ts">
	import { onMount } from 'svelte';
	import { fade } from 'svelte/transition';
	import * as m from '$paraglide/messages';
	import TableIcon from '~icons/ph/table';
	import Trash from '~icons/ph/trash';

	interface Props {
		isTableActive: boolean;
		canInsertTable: boolean;
		canAddRow: boolean;
		canAddColumn: boolean;
		canDeleteTable: boolean;
		onInsertTable: (rows: number, cols: number) => void;
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

	const tablePickerMax = 6;
	const tablePickerCellSize = 20;
	const tablePickerItems = Array.from({ length: tablePickerMax * tablePickerMax }, (_, index) => ({
		row: Math.floor(index / tablePickerMax) + 1,
		col: (index % tablePickerMax) + 1
	}));

	let menuElement: HTMLDivElement | null = null;
	let pickerOpen = $state(false);
	let hoveredRows = $state(0);
	let hoveredCols = $state(0);
	let manualRows = $state('3');
	let manualCols = $state('3');

	const activeToggleClass = 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900';
	const inactiveToggleClass =
		'text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800';
	const pillButtonBaseClass =
		'inline-flex h-8 shrink-0 items-center gap-1 rounded-md px-2 text-xs leading-none transition-colors disabled:cursor-not-allowed disabled:opacity-50';

	function handleSelect(rows: number, cols: number) {
		pickerOpen = false;
		onInsertTable(rows, cols);
	}

	function parsePositiveInt(value: string, fallback: number) {
		const parsed = Number.parseInt(value, 10);
		if (!Number.isFinite(parsed) || parsed < 1) {
			return fallback;
		}
		return parsed;
	}

	function handleManualInsert() {
		const rows = parsePositiveInt(manualRows, 3);
		const cols = parsePositiveInt(manualCols, 3);
		manualRows = String(rows);
		manualCols = String(cols);
		handleSelect(rows, cols);
	}

	onMount(() => {
		const handlePointerDown = (event: PointerEvent) => {
			if (!pickerOpen || !menuElement) return;
			const target = event.target;
			if (target instanceof Node && menuElement.contains(target)) return;
			pickerOpen = false;
		};

		document.addEventListener('pointerdown', handlePointerDown);
		return () => {
			document.removeEventListener('pointerdown', handlePointerDown);
		};
	});
</script>

<div bind:this={menuElement} class="relative shrink-0">
	<button
		type="button"
		title={m.editor_toolbar_insert_table()}
		aria-label={m.editor_toolbar_insert_table()}
		disabled={!canInsertTable}
		class={`${pillButtonBaseClass} ${isTableActive ? activeToggleClass : inactiveToggleClass}`}
		onclick={() => {
			if (!canInsertTable) return;
			hoveredRows = 0;
			hoveredCols = 0;
			pickerOpen = !pickerOpen;
		}}
	>
		<TableIcon class="h-4 w-4" />
	</button>

	{#if pickerOpen}
		<div
			in:fade={{ duration: 120 }}
			out:fade={{ duration: 100 }}
			class="absolute left-0 top-[calc(100%+0.4rem)] z-20 rounded-xl border border-zinc-200 bg-white p-2 shadow-xl shadow-zinc-900/10 dark:border-zinc-700 dark:bg-zinc-900 dark:shadow-black/30"
		>
			<div
				class="grid gap-1"
				style={`grid-template-columns: repeat(${tablePickerMax}, ${tablePickerCellSize}px); width: max-content;`}
			>
				{#each tablePickerItems as item}
					<button
						type="button"
						class={`rounded-[4px] border transition-colors ${
							item.row <= hoveredRows && item.col <= hoveredCols
								? 'border-zinc-900 bg-zinc-900 dark:border-zinc-100 dark:bg-zinc-100'
								: 'border-zinc-300 bg-zinc-50 hover:border-zinc-400 dark:border-zinc-700 dark:bg-zinc-800 dark:hover:border-zinc-600'
						}`}
						style={`width: ${tablePickerCellSize}px; height: ${tablePickerCellSize}px;`}
						aria-label={`${item.row} x ${item.col}`}
						onmouseenter={() => {
							hoveredRows = item.row;
							hoveredCols = item.col;
						}}
						onfocus={() => {
							hoveredRows = item.row;
							hoveredCols = item.col;
						}}
						onclick={() => handleSelect(item.row, item.col)}
					></button>
				{/each}
			</div>
			<p class="mt-2 whitespace-nowrap text-center text-xs text-zinc-500 dark:text-zinc-400">
				{hoveredRows > 0 && hoveredCols > 0 ? `${hoveredRows} x ${hoveredCols}` : '选择表格大小'}
			</p>
			<div class="mt-2 flex items-center gap-2">
				<label class="flex min-w-0 items-center gap-1 rounded-md border border-zinc-200 bg-zinc-50 px-2 py-1 text-xs text-zinc-600 dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-300">
					<span>{m.editor_table_rows_label()}</span>
					<input
						type="number"
						min="1"
						inputmode="numeric"
						class="w-10 bg-transparent text-center outline-none"
						bind:value={manualRows}
					/>
				</label>
				<label class="flex min-w-0 items-center gap-1 rounded-md border border-zinc-200 bg-zinc-50 px-2 py-1 text-xs text-zinc-600 dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-300">
					<span>{m.editor_table_columns_label()}</span>
					<input
						type="number"
						min="1"
						inputmode="numeric"
						class="w-10 bg-transparent text-center outline-none"
						bind:value={manualCols}
					/>
				</label>
				<button
					type="button"
					class="inline-flex h-8 shrink-0 items-center rounded-md bg-zinc-900 px-3 text-xs font-medium text-white transition-colors hover:bg-zinc-800 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
					onclick={handleManualInsert}
				>
					{m.editor_table_create_action()}
				</button>
			</div>
		</div>
	{/if}
</div>

{#if isTableActive}
	<div
		in:fade={{ duration: 120 }}
		out:fade={{ duration: 120 }}
		class="inline-flex shrink-0 items-center gap-1 rounded-lg px-1 outline outline-1 -outline-offset-1 outline-zinc-200 dark:outline-zinc-700"
	>
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
			<Trash class="h-4 w-4" />
		</button>
	</div>
{/if}

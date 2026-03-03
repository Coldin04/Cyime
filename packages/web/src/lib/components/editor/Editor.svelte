<script lang="ts">
	import { Crepe } from '@milkdown/crepe';
	import '@milkdown/crepe/theme/common/style.css';
	import '@milkdown/crepe/theme/frame-dark.css';
	import { replaceAll } from '@milkdown/utils';

	interface Props {
		content: string;
		onContentChange?: (content: string) => void;
	}

	let { content, onContentChange }: Props = $props();

	// 捕获内容的初始值，这是一个非响应式变量。
	const initialContent = content;

	let editorContainer: HTMLDivElement;
	let crepe: Crepe | null = null;
	let isCreated = false;
	// 使用 $state 来跟踪是否是 prop 更新，这样在闭包里也能拿到最新值
	let isUpdatingFromProp = $state(false);
	// 仍然需要 localContent 来防止更新循环。
	// 用 prop 的值来初始化它，以便它们在开始时保持一致。
	let localContent = $state(content);

	// 用于创建的 Effect，只运行一次。
	$effect(() => {
		if (!editorContainer) return;

		crepe = new Crepe({
			root: editorContainer,
			// 使用非响应式的初始值。
			defaultValue: initialContent
		});

		// 配置 listener 插件来监听 markdown 变化
		crepe.on((api) => {
			api.markdownUpdated((_, markdown: string) => {
				localContent = markdown;
				// 如果是从 prop 更新导致的，不要触发回调
				if (isUpdatingFromProp) return;
				onContentChange?.(markdown);
			});
		});

		// 等待编辑器创建完成
		crepe.create().then(() => {
			isCreated = true;
		});

		return () => {
			crepe?.destroy();
			crepe = null;
			isCreated = false;
		};
	});

	// 用于更新的 Effect。
	$effect(() => {
		if (!crepe || !isCreated) return;

		// 如果外部内容与内部内容不一致，则进行更新。
		// 这可以处理文档切换等情况，同时避免用户输入时的循环更新。
		if (content === localContent) return;

		isUpdatingFromProp = true;
		crepe.editor.action(replaceAll(content));
		// 使用 setTimeout 确保在下一个事件循环重置标志
		setTimeout(() => {
			isUpdatingFromProp = false;
		}, 0);
	});
</script>

<div class="milkdown-editor h-full w-full" bind:this={editorContainer}></div>

<style>
	/* 修复滚动问题 */
	.milkdown-editor {
	    min-height: 100%;
        overflow: auto;      
	}
	
</style>

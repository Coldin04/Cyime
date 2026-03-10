type ClickOutsideOptions = {
	enabled?: boolean;
	handler: () => void;
};

export function clickOutside(node: HTMLElement, options: ClickOutsideOptions) {
	let currentOptions = options;

	function handlePointerDown(event: PointerEvent) {
		if (!currentOptions.enabled) {
			return;
		}

		const target = event.target;
		if (target instanceof Node && !node.contains(target)) {
			currentOptions.handler();
		}
	}

	document.addEventListener('pointerdown', handlePointerDown, true);

	return {
		update(nextOptions: ClickOutsideOptions) {
			currentOptions = nextOptions;
		},
		destroy() {
			document.removeEventListener('pointerdown', handlePointerDown, true);
		}
	};
}

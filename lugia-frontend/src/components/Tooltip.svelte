<script lang="ts">
	import type { Snippet } from "svelte";

	let {
		content,
		children,
		position = "top",
		class: customClass = "",
		"data-testid": dataTestid
	}: {
		content: string;
		children: Snippet;
		position?: "top" | "bottom" | "left" | "right";
		class?: string;
		"data-testid"?: string;
	} = $props();

	let showTooltip = $state(false);

	const positionClasses = $derived(() => {
		switch (position) {
			case "top":
				return "bottom-full left-1/2 transform -translate-x-1/2 mb-2";
			case "bottom":
				return "top-full left-1/2 transform -translate-x-1/2 mt-2";
			case "left":
				return "right-full top-1/2 transform -translate-y-1/2 mr-2";
			case "right":
				return "left-full top-1/2 transform -translate-y-1/2 ml-2";
			default:
				return "bottom-full left-1/2 transform -translate-x-1/2 mb-2";
		}
	});

	const arrowClasses = $derived(() => {
		switch (position) {
			case "top":
				return "top-full left-1/2 transform -translate-x-1/2 border-l-transparent border-r-transparent border-b-transparent border-t-gray-800";
			case "bottom":
				return "bottom-full left-1/2 transform -translate-x-1/2 border-l-transparent border-r-transparent border-t-transparent border-b-gray-800";
			case "left":
				return "left-full top-1/2 transform -translate-y-1/2 border-t-transparent border-b-transparent border-r-transparent border-l-gray-800";
			case "right":
				return "right-full top-1/2 transform -translate-y-1/2 border-t-transparent border-b-transparent border-l-transparent border-r-gray-800";
			default:
				return "top-full left-1/2 transform -translate-x-1/2 border-l-transparent border-r-transparent border-b-transparent border-t-gray-800";
		}
	});
</script>

<div
	class="relative inline-block {customClass}"
	data-testid={dataTestid}
	onmouseenter={() => (showTooltip = true)}
	onmouseleave={() => (showTooltip = false)}
	onfocus={() => (showTooltip = true)}
	onblur={() => (showTooltip = false)}
	role="button"
	tabindex="0"
>
	{@render children()}

	{#if showTooltip}
		<div
			class="absolute z-50 px-3 py-2 text-sm text-white bg-gray-800 rounded-md shadow-lg whitespace-nowrap {positionClasses()}"
			role="tooltip"
		>
			{content}
			<div class="absolute w-0 h-0 border-4 {arrowClasses()}"></div>
		</div>
	{/if}
</div>

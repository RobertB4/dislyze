<script lang="ts">
	import type { Snippet } from "svelte";

	let {
		content,
		children,
		position = "top",
		class: customClass = "",
		"data-testid": dataTestid
	}: {
		content: Snippet;
		children: Snippet;
		position?: "top" | "bottom" | "left" | "right";
		class?: string;
		"data-testid"?: string;
	} = $props();

	let showTooltip = $state(false);
	let triggerElement: HTMLElement;
	let tooltipStyles = $state("");
	let arrowStyles = $state("");

	const updateTooltipPosition = () => {
		if (!triggerElement) return;

		const rect = triggerElement.getBoundingClientRect();
		const spacing = 8;
		const arrowSize = 6;

		let top: number, left: number;
		let arrowTop: number, arrowLeft: number;

		switch (position) {
			case "top":
				top = rect.top - spacing;
				left = rect.left + rect.width / 2;
				tooltipStyles = `top: ${top}px; left: ${left}px; transform: translate(-50%, -100%);`;

				arrowTop = rect.top - spacing / 2 - arrowSize / 2 - 1;
				arrowLeft = rect.left + rect.width / 2;
				arrowStyles = `top: ${arrowTop}px; left: ${arrowLeft}px; transform: translate(-50%, -50%) rotate(45deg);`;
				break;
			case "bottom":
				top = rect.bottom + spacing;
				left = rect.left + rect.width / 2;
				tooltipStyles = `top: ${top}px; left: ${left}px; transform: translate(-50%, 0);`;

				arrowTop = rect.bottom + spacing / 2 + arrowSize / 2 + 1;
				arrowLeft = rect.left + rect.width / 2;
				arrowStyles = `top: ${arrowTop}px; left: ${arrowLeft}px; transform: translate(-50%, -50%) rotate(225deg);`;
				break;
			case "left":
				top = rect.top + rect.height / 2;
				left = rect.left - spacing;
				tooltipStyles = `top: ${top}px; left: ${left}px; transform: translate(-100%, -50%);`;
				// Arrow points right, positioned to overlap with right edge of tooltip
				arrowTop = rect.top + rect.height / 2;
				arrowLeft = rect.left - spacing / 2 - arrowSize / 2 - 1;
				arrowStyles = `top: ${arrowTop}px; left: ${arrowLeft}px; transform: translate(-50%, -50%) rotate(315deg);`;
				break;
			case "right":
				top = rect.top + rect.height / 2;
				left = rect.right + spacing;
				tooltipStyles = `top: ${top}px; left: ${left}px; transform: translate(0, -50%);`;

				arrowTop = rect.top + rect.height / 2;
				arrowLeft = rect.right + spacing / 2 + arrowSize / 2 + 1;
				arrowStyles = `top: ${arrowTop}px; left: ${arrowLeft}px; transform: translate(-50%, -50%) rotate(135deg);`;
				break;
			default:
				top = rect.top - spacing;
				left = rect.left + rect.width / 2;
				tooltipStyles = `top: ${top}px; left: ${left}px; transform: translate(-50%, -100%);`;

				arrowTop = rect.top - spacing / 2 - arrowSize / 2 - 1;
				arrowLeft = rect.left + rect.width / 2;
				arrowStyles = `top: ${arrowTop}px; left: ${arrowLeft}px; transform: translate(-50%, -50%) rotate(45deg);`;
		}
	};

	const handleMouseEnter = () => {
		showTooltip = true;
		updateTooltipPosition();
	};
</script>

<div
	bind:this={triggerElement}
	class="relative inline-block {customClass}"
	data-testid={dataTestid}
	onmouseenter={handleMouseEnter}
	onmouseleave={() => (showTooltip = false)}
	onfocus={handleMouseEnter}
	onblur={() => (showTooltip = false)}
	role="button"
	tabindex="0"
>
	{@render children()}
</div>

{#if showTooltip}
	<div
		class="fixed z-50 px-3 py-2 text-sm text-white bg-gray-800 rounded-md shadow-lg max-w-xs"
		style={tooltipStyles}
		role="tooltip"
	>
		{@render content()}
	</div>
	<div class="fixed z-50 w-2 h-2 bg-gray-800" style={arrowStyles}></div>
{/if}

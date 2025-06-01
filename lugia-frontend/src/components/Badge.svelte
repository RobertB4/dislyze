<script lang="ts">
	import { assertNever } from "$lib/assertNever";
	import type { Snippet } from "svelte";

	type ColorPalette = "green" | "yellow" | "red" | "orange" | "gray";

	let {
		class: customClass = "",
		color,
		children,
		"data-testid": dataTestid
	}: {
		class?: string;
		color: ColorPalette;
		children: Snippet;
		"data-testid"?: string;
	} = $props();

	const colorClass = $derived(() => {
		switch (color) {
			case "green":
				return "bg-green-100 text-green-800";
			case "red":
				return "bg-red-100 text-red-800";
			case "yellow":
				return "bg-yellow-100 text-yellow-800";
			case "orange":
				return "bg-orange-100 text-orange-800";
			case "gray":
				return "bg-gray-100 text-gray-800";
			default:
				throw assertNever(color);
		}
	});
</script>

<span
	role="menu"
	tabindex={0}
	class="inline-flex rounded-full px-3 py-1 text-xs font-semibold leading-5 {colorClass()} {customClass}"
	data-testid={dataTestid}
	>{@render children()}</span
>

<script lang="ts">
	import { assertNever } from "$lib/assertNever";
	import type { Snippet } from "svelte";

	type ColorPalette = "green" | "yellow" | "red" | "orange" | "gray" | "blue";

	let {
		class: customClass = "",
		color,
		size = "md",
		rounded = "full",
		children,
		"data-testid": dataTestid
	}: {
		class?: string;
		color: ColorPalette;
		size?: "sm" | "md" | "lg";
		rounded?: "full" | "md";
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
			case "blue":
				return "bg-blue-100 text-blue-800";
			default:
				throw assertNever(color);
		}
	});

	const sizeClass = $derived(() => {
		switch (size) {
			case "sm":
				return "px-2 py-1 text-xs";
			case "md":
				return "px-3 py-1 text-xs";
			case "lg":
				return "px-4 py-2 text-sm";
			default:
				throw assertNever(size);
		}
	});

	const roundedClass = $derived(() => {
		switch (rounded) {
			case "full":
				return "rounded-full";
			case "md":
				return "rounded-md";
			default:
				throw assertNever(rounded);
		}
	});
</script>

<span
	role="menu"
	tabindex={0}
	class="inline-flex {roundedClass()} {sizeClass()} font-semibold leading-5 {colorClass()} {customClass}"
	data-testid={dataTestid}>{@render children()}</span
>

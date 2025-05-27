<script lang="ts">
	import type { Snippet } from "svelte";

	type ButtonType = "button" | "submit" | "reset";
	type ButtonVariant = "primary" | "secondary" | "danger" | "link";

	let {
		type = "button" as ButtonType,
		disabled = false,
		loading = false,
		variant = "primary" as ButtonVariant,
		fullWidth = false,
		class: customClass = "",
		onclick,
		children,
		dataTestId = undefined
	}: {
		type?: ButtonType;
		disabled?: boolean;
		loading?: boolean;
		variant?: ButtonVariant;
		fullWidth?: boolean;
		class?: string;
		onclick?: (event: MouseEvent) => void;
		children: Snippet;
		dataTestId?: string;
	} = $props();

	const baseStyles = $derived(
		variant === "link"
			? "inline-flex items-center justify-center text-sm font-medium cursor-pointer focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed"
			: "group relative flex justify-center cursor-pointer py-2 px-4 border text-sm font-medium rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
	);

	const variantStyles = $derived({
		primary:
			"border-transparent text-white bg-orange-600 hover:bg-orange-700 focus:ring-orange-500",
		secondary: "border-gray-300 text-gray-700 bg-white hover:bg-gray-50 focus:ring-orange-500",
		danger: "border-transparent text-white bg-red-600 hover:bg-red-700 focus:ring-red-500",
		link: "border-transparent text-indigo-600 hover:text-indigo-800 hover:underline focus:ring-indigo-500 p-0"
	});

	const widthClass = $derived(fullWidth ? "w-full" : "");

	const buttonClass = $derived(
		`${baseStyles} ${variantStyles[variant]} ${widthClass} ${customClass}`
	);
</script>

<button {type} {disabled} class={buttonClass} {onclick} data-testid={dataTestId}>
	{#if loading && variant !== "link"}
		<span class="absolute left-0 inset-y-0 flex items-center pl-3">
			<svg
				class="animate-spin h-5 w-5 {variant === 'primary' || variant === 'danger'
					? 'text-white'
					: 'text-orange-600'}"
				xmlns="http://www.w3.org/2000/svg"
				fill="none"
				viewBox="0 0 24 24"
			>
				<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
				></circle>
				<path
					class="opacity-75"
					fill="currentColor"
					d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
				></path>
			</svg>
		</span>
	{/if}
	{#if loading && variant === "link"}
		<svg
			class="animate-spin h-5 w-5 mr-2 text-indigo-600"
			xmlns="http://www.w3.org/2000/svg"
			fill="none"
			viewBox="0 0 24 24"
		>
			<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
			></circle>
			<path
				class="opacity-75"
				fill="currentColor"
				d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
			></path>
		</svg>
	{/if}
	{@render children()}
</button>

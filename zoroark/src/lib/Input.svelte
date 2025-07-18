<script lang="ts">
	let {
		type = "text" as "text" | "email" | "password" | "number" | "tel" | "url",
		id,
		name,
		label,
		placeholder = "",
		required = false,
		disabled = false,
		error,
		class: customClass = "",
		value = $bindable("" as string | number),
		variant = "default" as "default" | "underlined",
		oninput,
		"data-testid": dataTestid
	}: {
		type?: "text" | "email" | "password" | "number" | "tel" | "url";
		id: string;
		name: string;
		label: string;
		placeholder?: string;
		required?: boolean;
		disabled?: boolean;
		error?: string | undefined;
		class?: string;
		value?: string | number;
		variant?: "default" | "underlined";
		oninput?: (event: Event) => void;
		"data-testid"?: string;
	} = $props();

	const variantStyles = {
		default:
			"appearance-none rounded-md relative block w-full px-3 py-2 border text-gray-900 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm",
		underlined:
			"appearance-none bg-transparent block w-full px-1 py-2 border-x-0 border-t-0 border-b-2 text-gray-900 focus:outline-none focus:ring-0 sm:text-sm"
	};

	const stateStyles = {
		default: {
			default: "border-gray-300 placeholder-gray-500",
			error:
				"border-red-300 placeholder-red-300 text-red-900 focus:border-red-500 focus:ring-red-500",
			disabled: "border-gray-300 bg-gray-100 cursor-not-allowed"
		},
		underlined: {
			default: "border-gray-300 placeholder-gray-500 focus:border-indigo-600",
			error: "border-red-500 placeholder-red-400 text-red-700 focus:border-red-600",
			disabled: "border-gray-300 bg-transparent cursor-not-allowed opacity-50"
		}
	};

	const inputClass = $derived(
		`${variantStyles[variant]} ${error ? stateStyles[variant].error : stateStyles[variant].default} ${disabled ? stateStyles[variant].disabled : ""} ${customClass}`
	);
</script>

<div>
	<label for={id} class="sr-only">{label}</label>
	<input
		{id}
		{name}
		{type}
		{required}
		{disabled}
		{placeholder}
		{value}
		{oninput}
		class={inputClass}
		data-testid={dataTestid}
	/>
	{#if error}
		<p data-testid={`${id}-error`} class="mt-1 text-sm text-red-600">{error}</p>
	{/if}
</div>

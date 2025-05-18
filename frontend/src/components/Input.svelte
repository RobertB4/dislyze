<script lang="ts">
	export let type: "text" | "email" | "password" | "number" | "tel" | "url" = "text";
	export let id: string;
	export let name: string;
	export let label: string;
	export let placeholder = "";
	export let required = false;
	export let disabled = false;
	export let error: string | undefined = undefined;
	export let className = "";
	export let value: string | number = "";

	const baseStyles =
		"appearance-none rounded-md relative block w-full px-3 py-2 border text-gray-900 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm";
	const stateStyles = {
		default: "border-gray-300 placeholder-gray-500",
		error: "border-red-300 placeholder-red-300",
		disabled: "bg-gray-100 cursor-not-allowed"
	};

	$: inputClass = `${baseStyles} ${error ? stateStyles.error : stateStyles.default} ${
		disabled ? stateStyles.disabled : ""
	} ${className}`;
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
		class={inputClass}
		on:input
		on:change
		on:blur
		on:focus
	/>
	{#if error}
		<p class="mt-1 text-sm text-red-600">{error}</p>
	{/if}
</div>

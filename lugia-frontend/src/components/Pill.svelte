<script lang="ts">
	let {
		selected = false,
		onclick,
		disabled = false,
		variant,
		size = "md",
		class: className = "",
		children,
		...rest
	}: {
		selected?: boolean;
		onclick?: () => void;
		disabled?: boolean;
		variant: "orange" | "blue" | "green" | "red" | "gray";
		size?: "sm" | "md" | "lg";
		class?: string;
		children: any;
		[key: string]: any;
	} = $props();

	// Base styles
	const baseStyles =
		"inline-flex items-center justify-center font-medium rounded-full border-2 transition-all duration-200 cursor-pointer hover:shadow-sm focus:outline-none focus:ring-2 focus:ring-offset-2";

	// Size variants
	const sizeStyles = {
		sm: "px-2 py-1 text-xs",
		md: "px-3 py-1 text-sm",
		lg: "px-4 py-2 text-base"
	};

	// Color variants - selected and unselected states
	const variantStyles = {
		orange: {
			selected:
				"border-orange-500 bg-orange-500 text-white hover:bg-orange-600 focus:ring-orange-500",
			unselected:
				"border-orange-300 bg-white text-orange-700 hover:border-orange-400 hover:bg-orange-50 focus:ring-orange-500"
		},
		blue: {
			selected: "border-blue-500 bg-blue-500 text-white hover:bg-blue-600 focus:ring-blue-500",
			unselected:
				"border-blue-300 bg-white text-blue-700 hover:border-blue-400 hover:bg-blue-50 focus:ring-blue-500"
		},
		green: {
			selected: "border-green-500 bg-green-500 text-white hover:bg-green-600 focus:ring-green-500",
			unselected:
				"border-green-300 bg-white text-green-700 hover:border-green-400 hover:bg-green-50 focus:ring-green-500"
		},
		red: {
			selected: "border-red-500 bg-red-500 text-white hover:bg-red-600 focus:ring-red-500",
			unselected:
				"border-red-300 bg-white text-red-700 hover:border-red-400 hover:bg-red-50 focus:ring-red-500"
		},
		gray: {
			selected: "border-gray-500 bg-gray-500 text-white hover:bg-gray-600 focus:ring-gray-500",
			unselected:
				"border-gray-300 bg-white text-gray-700 hover:border-gray-400 hover:bg-gray-50 focus:ring-gray-500"
		}
	};

	// Disabled styles
	const disabledStyles = "opacity-50 cursor-not-allowed hover:shadow-none";

	// Combine all styles
	let computedStyles = $derived(() => {
		const styles = [
			baseStyles,
			sizeStyles[size],
			disabled ? disabledStyles : variantStyles[variant][selected ? "selected" : "unselected"],
			className
		]
			.filter(Boolean)
			.join(" ");

		return styles;
	});
</script>

<button
	type="button"
	class={computedStyles()}
	{onclick}
	{disabled}
	data-selected={selected}
	{...rest}
>
	{@render children()}
</button>

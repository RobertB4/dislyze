<script lang="ts">
	import type { Snippet } from "svelte";

	type AlertType = "danger" | "warning" | "info" | "success";

	let {
		type = "info" as AlertType,
		title = "" as string,
		children,
		"data-testid": dataTestid
	}: {
		type?: AlertType;
		title?: string;
		children: Snippet;
		"data-testid"?: string;
	} = $props();

	const baseClasses = "rounded-md p-4";
	const typeClasses = $derived({
		danger: "bg-red-50 text-red-700",
		warning: "bg-yellow-50 text-yellow-700",
		info: "bg-blue-50 text-blue-700",
		success: "bg-green-50 text-green-700"
	});
</script>

<div class="{baseClasses} {typeClasses[type]}" data-testid={dataTestid}>
	<div class="flex items-center">
		<div class="flex-shrink-0">
			{#if type === "danger"}
				<!-- Heroicon name: exclamation-triangle -->
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="1.5"
					stroke="currentColor"
					class="size-8"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z"
					/>
				</svg>
			{:else if type === "warning"}
				<!-- Heroicon name: shield-exclamation -->
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="1.5"
					stroke="currentColor"
					class="size-8"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M12 9v3.75m0-10.036A11.959 11.959 0 0 1 3.598 6 11.99 11.99 0 0 0 3 9.75c0 5.592 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.57-.598-3.75h-.152c-3.196 0-6.1-1.25-8.25-3.286Zm0 13.036h.008v.008H12v-.008Z"
					/>
				</svg>
			{:else if type === "info"}
				<!-- Heroicon name: information-circle -->
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="1.5"
					stroke="currentColor"
					class="size-8"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="m11.25 11.25.041-.02a.75.75 0 0 1 1.063.852l-.708 2.836a.75.75 0 0 0 1.063.853l.041-.021M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9-3.75h.008v.008H12V8.25Z"
					/>
				</svg>
			{:else if type === "success"}
				<!-- Heroicon name: check-circle -->
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="1.5"
					stroke="currentColor"
					class="size-8"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M9 12.75 11.25 15 15 9.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z"
					/>
				</svg>
			{/if}
		</div>
		<div class="ml-3">
			{#if title}
				<h3
					class="text-sm font-medium {typeClasses[type]}"
					data-testid={dataTestid ? `${dataTestid}-title` : undefined}
				>
					{title}
				</h3>
			{/if}
			<div class="text-sm {typeClasses[type]}">
				{@render children()}
			</div>
		</div>
	</div>
</div>

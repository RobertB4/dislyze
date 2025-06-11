<script lang="ts">
	import { onMount } from "svelte";
	import { fade, slide } from "svelte/transition";

	type ToastMode = "success" | "error" | "info";

	let {
		text,
		mode = "info" as ToastMode,
		onClose,
		autocloseDuration = 5000,
		"data-testid": dataTestId
	}: {
		text: string;
		mode?: ToastMode;
		onClose: () => void;
		autocloseDuration?: number;
		"data-testid"?: string;
	} = $props();

	let visible = $state(false);

	onMount(() => {
		visible = true;

		if (autocloseDuration > 0) {
			const timer = setTimeout(() => {
				closeToast();
			}, autocloseDuration);
			return () => clearTimeout(timer);
		}
	});

	function closeToast() {
		visible = false;
		// Wait for fade out animation before calling onClose
		setTimeout(onClose, 301);
	}
</script>

{#if visible}
	<div
		data-testid={dataTestId}
		class="fixed left-4 bottom-4 z-50 min-w-[300px] max-w-[400px] rounded-lg shadow-lg"
		transition:slide={{ duration: 300 }}
		role="alert"
		aria-live="assertive"
	>
		<div
			class="relative flex items-center justify-between rounded-lg p-4 text-white {mode ===
			'success'
				? 'bg-green-600'
				: mode === 'error'
					? 'bg-red-600'
					: 'bg-blue-600'}"
			transition:fade={{ duration: 300 }}
		>
			<p class="text-sm font-medium">{text}</p>
			<button
				type="button"
				class="ml-4 inline-flex h-5 w-5 flex-shrink-0 cursor-pointer rounded-md p-0.5 text-white hover:bg-white/10 focus:outline-none focus:ring-2 focus:ring-white/20"
				onclick={closeToast}
				aria-label="閉じる"
				data-testid={dataTestId ? `${dataTestId}-close` : undefined}
			>
				<span class="sr-only">Close</span>
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M6 18L18 6M6 6l12 12"
					/>
				</svg>
			</button>
		</div>
	</div>
{/if}

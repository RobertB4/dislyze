<script lang="ts">
	import { onMount } from 'svelte';
	import { fade, slide } from 'svelte/transition';

	export let text: string;
	export let mode: 'success' | 'error' | 'info' = 'info';
	export let onClose: () => void;

	let visible = true;

	onMount(() => {
		const timer = setTimeout(() => {
			visible = false;
			setTimeout(onClose, 300); // Wait for fade out animation
		}, 5000);

		return () => clearTimeout(timer);
	});

	function handleClose() {
		visible = false;
		setTimeout(onClose, 300);
	}
</script>

{#if visible}
	<div
		class="fixed left-4 top-4 z-50 min-w-[300px] max-w-[400px] rounded-lg shadow-lg"
		transition:slide={{ duration: 300 }}
	>
		<div
			class="relative flex items-center justify-between rounded-lg p-4 text-white"
			class:bg-green-600={mode === 'success'}
			class:bg-red-600={mode === 'error'}
			class:bg-blue-600={mode === 'info'}
			transition:fade={{ duration: 300 }}
		>
			<p class="text-sm font-medium">{text}</p>
			<button
				type="button"
				class="ml-4 inline-flex h-5 w-5 flex-shrink-0 rounded-md p-0.5 text-white hover:bg-white/10 focus:outline-none focus:ring-2 focus:ring-white/20"
				on:click={handleClose}
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

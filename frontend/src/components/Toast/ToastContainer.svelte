<script lang="ts">
	import { toast } from "./toast";
	import Toast from "./Toast.svelte";
</script>

<div class="fixed left-0 top-0 z-50 flex flex-col gap-2 p-4">
	{#each $toast as t (t.id)}
		<Toast
			text={t.text}
			mode={t.mode}
			onClose={() => toast.remove(t.id)}
			data-testid={`toast-${t.id}`}
		/>
	{/each}
</div>

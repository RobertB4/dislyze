<script lang="ts">
	import "../app.css";
	import ToastContainer from "$components/Toast/ToastContainer.svelte";
	import { meCache } from "$lib/meCache";
	import type { LayoutData } from "./$types";
	import type { Snippet } from "svelte";

	let { data, children }: { data: LayoutData; children: Snippet } = $props();

	$effect(() => {
		if (data.me) {
			meCache.set(data.me);
		}
	});
</script>

<ToastContainer />
{@render children()}

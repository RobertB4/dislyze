<script lang="ts">
	import "../app.css";
	import ToastContainer from "$components/Toast/ToastContainer.svelte";
	import { page } from "$app/state";
	import { setMe } from "$lib/stores/meStore";
	import type { LayoutData } from "./$types";
	import type { Snippet } from "svelte";

	let { data, children }: { data: LayoutData; children: Snippet } = $props();

	// Effect to update the meStore based on initialUser from +layout.ts load function
	// and clear it if a page error occurs.
	$effect(() => {
		if (data.initialUser) {
			setMe(data.initialUser);
		}
	});
</script>

<ToastContainer />
{@render children()}

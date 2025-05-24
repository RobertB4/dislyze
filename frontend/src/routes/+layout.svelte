<script lang="ts">
	import "../app.css";
	import ToastContainer from "$components/Toast/ToastContainer.svelte";
	import { setMe } from "$lib/stores/meStore";
	import type { LayoutData } from "./$types";
	import type { Snippet } from "svelte";

	let { data, children }: { data: LayoutData; children: Snippet } = $props();

	$effect(() => {
		if (data.initialUser) {
			setMe(data.initialUser);
		}
	});
</script>

<ToastContainer />
{@render children()}

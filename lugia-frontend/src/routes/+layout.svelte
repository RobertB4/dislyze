<script lang="ts">
	import "../app.css";
	import { ToastContainer } from "@dislyze/zoroark";
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

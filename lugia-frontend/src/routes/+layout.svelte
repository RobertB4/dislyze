<script lang="ts">
	import "$lugia/app.css";
	import ToastContainer from "@dislyze/zoroark/ToastContainer";
	import { meCache } from "@dislyze/zoroark/meCache";
	import type { LayoutData } from "./$types";
	import type { Snippet } from "svelte";

	let { data, children }: { data: LayoutData; children: Snippet } = $props();

	// eslint-disable-next-line no-restricted-syntax
	$effect(() => {
		if (data.me) {
			meCache.set(data.me);
		}
	});
</script>

<ToastContainer />
{@render children()}

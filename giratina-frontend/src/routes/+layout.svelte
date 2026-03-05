<script lang="ts">
	import ToastContainer from "@dislyze/zoroark/ToastContainer";
	import "$giratina/app.css";
	import { meCache } from "@dislyze/zoroark/meCache";
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

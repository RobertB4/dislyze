<script lang="ts">
	import "../app.css";
	import ToastContainer from "$components/Toast/ToastContainer.svelte";
	import { me } from "$lib/me";
	import type { LayoutData } from "./$types";
	import type { Snippet } from "svelte";

	let { data, children }: { data: LayoutData; children: Snippet } = $props();

	$effect(() => {
		if (data.initialUser) {
			me.set(data.initialUser);
		}
	});
</script>

<ToastContainer />
{@render children()}

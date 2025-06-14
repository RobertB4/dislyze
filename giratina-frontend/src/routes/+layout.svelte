<script lang="ts">
	import { ToastContainer } from "@dislyze/zoroark";
	import "../app.css";
	import type { Snippet } from "svelte";

	let { children }: { children: Snippet } = $props();
</script>

<ToastContainer />
{@render children()}

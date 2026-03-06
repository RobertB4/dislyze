<!-- Feature doc: docs/features/authentication.md -->
<script lang="ts">
	import Layout from "$lugia/components/Layout.svelte";
	import type { PageData } from "./$types";

	const { data: pageData }: { data: PageData } = $props();
</script>

<Layout me={pageData.me} pageTitle="Dashboard">
	<h1>SvelteKit Frontend</h1>
</Layout>

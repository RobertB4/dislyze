<script lang="ts">
	import Layout from "$components/Layout.svelte";
	import { Button, Input, Badge, Alert } from "@dislyze/zoroark";
	import type { PageData } from "./$types";

	let { data: pageData }: { data: PageData } = $props();
</script>

<Layout pageTitle="テナント一覧" me={pageData.me}>
	<h1 class="px-4">Giratina Frontend</h1>
	<Button>testo2</Button>
	<Input id="asd" name="asd" label="ASD" />
	<Badge color="yellow">BADGE</Badge>
	<Alert>OH NO!</Alert>
</Layout>

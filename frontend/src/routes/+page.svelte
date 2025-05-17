<script lang="ts">
	import type { PageData } from './types';
	import { PUBLIC_API_URL } from '$env/static/public';
	import Layout from '$components/Layout.svelte';

	export let data: PageData;
</script>

<Layout>
	<h1>SvelteKit Frontend</h1>
	<div class="message-container">
		<h2>Message from Backend:</h2>
		<p class="backend-message">{data.message}</p>
	</div>
	<p class="api-info">Backend API: <a href={PUBLIC_API_URL}>{PUBLIC_API_URL}</a></p>
</Layout>

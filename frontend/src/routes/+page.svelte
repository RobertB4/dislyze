<script lang="ts">
	import type { PageData } from './types';
	import { PUBLIC_API_URL } from '$env/static/public';

	export let data: PageData;
</script>

<main>
	<h1>SvelteKit Frontend</h1>
	<div class="message-container">
		<h2>Message from Backend:</h2>
		<p class="backend-message">{data.message}</p>
	</div>
	<p class="api-info">Backend API: <a href={PUBLIC_API_URL}>{PUBLIC_API_URL}</a></p>
</main>

<style>
	main {
		padding: 2rem;
		max-width: 800px;
		margin: 0 auto;
		text-align: center;
	}

	h1 {
		color: #ff3e00;
		font-size: 2.5rem;
		margin-bottom: 1rem;
	}

	.message-container {
		margin: 2rem 0;
		padding: 1rem;
		background: #f9f9f9;
		border-radius: 8px;
		box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
	}

	h2 {
		color: #2c3e50;
		font-size: 1.5rem;
		margin-bottom: 0.5rem;
	}

	.backend-message {
		font-size: 1.2rem;
		color: #34495e;
		margin: 1rem 0;
	}

	.api-info {
		color: #7f8c8d;
		font-size: 0.9rem;
	}

	a {
		color: #4a9eff;
		text-decoration: none;
	}

	a:hover {
		text-decoration: underline;
	}
</style>

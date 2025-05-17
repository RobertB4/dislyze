<script lang="ts">
	import Layout from '$components/Layout.svelte';
	import type { PageData } from './$types';
	export let data: PageData;
</script>

<Layout>
	{#if data.error}
		<div class="bg-red-50 border border-red-200 rounded-lg p-4">
			<h2 class="text-red-800 font-semibold">Error</h2>
			<p class="text-red-600">{data.error}</p>
		</div>
	{:else}
		<div class="bg-white shadow rounded-lg">
			<div class="px-4 py-5 sm:p-6">
				<h2 class="text-lg font-medium text-gray-900">Users</h2>
				<div class="mt-4">
					<pre class="bg-gray-50 p-4 rounded-lg overflow-auto text-sm">
						{JSON.stringify(data.users, null, 2)}
					</pre>
				</div>
			</div>
		</div>
	{/if}
</Layout>

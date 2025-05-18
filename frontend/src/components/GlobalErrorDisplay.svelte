<script lang="ts">
	import { errorStore } from "$lib/errors";
	import { goto } from "$app/navigation";
	import { PUBLIC_API_URL } from "$env/static/public";

	async function handleReturnToLogin() {
		try {
			await fetch(`${PUBLIC_API_URL}/auth/logout`, {
				method: "POST",
				credentials: "include"
			});
		} catch (logoutError) {
			console.error("Logout attempt from error page failed:", logoutError);
		}
		errorStore.clearError();
		goto("/auth/login");
	}

	function handleGoHome() {
		errorStore.clearError();
		goto("/");
	}
</script>

{#if $errorStore.statusCode}
	<div
		class="fixed inset-0 z-50 flex flex-col items-center justify-center bg-gray-100 bg-opacity-90 p-4 text-center"
	>
		<h1 class="mb-4 text-4xl font-bold text-red-500">
			エラーが発生しました ({$errorStore.statusCode})
		</h1>
		<p class="mb-8 text-lg">
			{$errorStore.message || "予期せぬエラーが発生しました。"}
		</p>
		<div class="flex space-x-4">
			<button
				on:click={handleReturnToLogin}
				class="rounded bg-blue-600 px-6 py-3 text-lg cursor-pointer font-semibold text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-50 transition-colors duration-150"
			>
				ログイン画面へ戻る
			</button>
			<button
				on:click={handleGoHome}
				class="rounded bg-gray-600 px-6 py-3 text-lg cursor-pointer font-semibold text-white hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-gray-500 focus:ring-opacity-50 transition-colors duration-150"
			>
				ダッシュボードへ
			</button>
		</div>
	</div>
{/if}

<style>
	/* Basic Tailwind styling is used inline. Add any specific styles here if needed. */
</style>

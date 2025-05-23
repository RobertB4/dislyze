<script lang="ts">
	import { page } from "$app/state";
	import { goto } from "$app/navigation";
	import { PUBLIC_API_URL } from "$env/static/public";
	import Button from "$components/Button.svelte";

	async function handleReturnToLogin() {
		try {
			await fetch(`${PUBLIC_API_URL}/auth/logout`, {
				method: "POST",
				credentials: "include"
			});
		} catch (logoutError) {
			console.error("Logout attempt from error page failed:", logoutError);
		}
		goto("/auth/login");
	}

	function handleGoHome() {
		goto("/");
	}
</script>

{#if page.error}
	<div
		class="fixed inset-0 z-50 flex flex-col items-center justify-center bg-base-100 bg-opacity-90 p-4 text-center backdrop-blur-sm"
	>
		<h1 class="mb-4 text-4xl font-bold text-red-500">
			エラーが発生しました ({page.status})
		</h1>
		<p class="mb-8 text-lg text-base-content">
			{page.error.message || "予期せぬエラーが発生しました。"}
		</p>
		<div class="flex space-x-4">
			<Button onclick={handleGoHome} class="btn btn-neutral btn-lg cursor-pointer">
				トップページへ戻る
			</Button>
			<Button variant="secondary" onclick={handleReturnToLogin} class="btn btn-primary btn-lg">
				ログイン画面へ戻る
			</Button>
		</div>
	</div>
{/if}

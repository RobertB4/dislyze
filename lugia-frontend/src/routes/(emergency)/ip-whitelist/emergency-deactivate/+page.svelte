<script lang="ts">
	import type { PageData } from "./$types";

	// TODO: fundamentally rework this page. claude hallucinated a bunch of stuff.
	let { data }: { data: PageData } = $props();
</script>

<main class="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
	<div class="max-w-md w-full space-y-8">
		<div class="text-center">
			<a href="/">
				<img src="/logofull.png" alt="Dislyze Logo" class="mx-auto h-12 w-auto" />
			</a>
			<h2 class="mt-6 text-center text-3xl font-extrabold text-gray-900">IP制限の緊急解除</h2>
		</div>

		<div class="mt-8 space-y-6">
			{#if data.success}
				<!-- Success state -->
				<div class="rounded-md bg-green-50 p-4">
					<div class="flex">
						<div class="flex-shrink-0">
							<svg class="h-5 w-5 text-green-400" viewBox="0 0 20 20" fill="currentColor">
								<path
									fill-rule="evenodd"
									d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
									clip-rule="evenodd"
								/>
							</svg>
						</div>
						<div class="ml-3">
							<h3 class="text-sm font-medium text-green-800">IP制限が正常に解除されました</h3>
							<div class="mt-2 text-sm text-green-700">
								<p>
									IP制限による
									アクセス制限が解除されました。アプリケーションに再度アクセスできるようになりました。
								</p>
							</div>
						</div>
					</div>
				</div>

				<div class="text-center">
					<p class="text-sm text-gray-600 mb-4">
						3秒後に自動的にログインページにリダイレクトします...
					</p>
					<a
						href="/auth/login"
						class="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
					>
						すぐにログインページに移動
					</a>
				</div>
			{:else}
				<!-- This should not be reached as errors are handled in the load function -->
				<div class="rounded-md bg-red-50 p-4">
					<div class="flex">
						<div class="flex-shrink-0">
							<svg class="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
								<path
									fill-rule="evenodd"
									d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
									clip-rule="evenodd"
								/>
							</svg>
						</div>
						<div class="ml-3">
							<h3 class="text-sm font-medium text-red-800">緊急解除に失敗しました</h3>
							<div class="mt-2 text-sm text-red-700">
								<p>予期しないエラーが発生しました。お手数ですが、管理者にお問い合わせください。</p>
							</div>
						</div>
					</div>
				</div>
			{/if}
		</div>

		<div class="text-sm text-center">
			<a href="/auth/login" class="font-medium text-indigo-600 hover:text-indigo-500">
				ログインページに戻る
			</a>
		</div>
	</div>
</main>

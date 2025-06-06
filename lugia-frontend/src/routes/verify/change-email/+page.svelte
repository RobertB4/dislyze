<script lang="ts">
	import Button from "$components/Button.svelte";
	import Alert from "$components/Alert.svelte";
	import type { PageData } from "./$types";
	import { goto } from "$app/navigation";

	let { data }: { data: PageData } = $props();

	const goToLogin = () => {
		const returnUrl = encodeURIComponent(
			`/verify/change-email?token=${encodeURIComponent(data.token)}`
		);
		goto(`/auth/login?redirect=${returnUrl}`);
	};

	const goToProfile = () => {
		goto("/settings/profile");
	};
</script>

<div class="min-h-screen bg-gray-50 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
	<div class="sm:mx-auto sm:w-full sm:max-w-md">
		<div class="bg-white py-8 px-4 shadow sm:rounded-lg sm:px-10">
			<div class="sm:mx-auto sm:w-full sm:max-w-md mb-6">
				<h2 class="text-center text-3xl font-extrabold text-gray-900" data-testid="verify-email-heading">メールアドレスの変更確認</h2>
			</div>

			{#if data.needsLogin}
				<div class="space-y-6" data-testid="needs-login-section">
					<Alert type="info">
						<p class="text-sm" data-testid="needs-login-message">メールアドレスの変更を完了するには、ログインする必要があります。</p>
					</Alert>

					<p class="text-sm text-gray-600 text-center" data-testid="login-instruction">
						アカウントにログイン後、メールアドレスの変更が自動的に完了します。
					</p>

					<div class="flex flex-col space-y-3">
						<Button variant="primary" onclick={goToLogin} class="w-full" data-testid="go-to-login-button">
							ログインしてメールアドレスを変更
						</Button>
					</div>
				</div>
			{:else if data.verificationFailed}
				<div class="space-y-6" data-testid="verification-failed-section">
					<Alert type="danger">
						<p class="text-sm" data-testid="verification-failed-message">
							メールアドレスの変更に失敗しました。
						</p>
						<p class="text-sm mt-1" data-testid="verification-failed-detail">
							リンクが無効または期限切れの可能性があります。
						</p>
					</Alert>

					<p class="text-sm text-gray-600 text-center" data-testid="retry-instruction">
						新しいメールアドレスの変更をお試しください。
					</p>

					<Button variant="primary" onclick={goToProfile} class="w-full" data-testid="back-to-profile-button">
						プロフィール設定に戻る
					</Button>
				</div>
			{/if}
		</div>
	</div>
</div>

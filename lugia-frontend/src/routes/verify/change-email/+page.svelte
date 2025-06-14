<script lang="ts">
	import { Button, Alert } from "@dislyze/zoroark";
	import type { PageData } from "./$types";
	import { goto } from "$app/navigation";

	let { data }: { data: PageData } = $props();

	const goToProfile = () => {
		goto("/settings/profile");
	};
</script>

<div class="min-h-screen bg-gray-50 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
	<div class="sm:mx-auto sm:w-full sm:max-w-md">
		<div class="bg-white py-8 px-4 shadow sm:rounded-lg sm:px-10">
			<div class="sm:mx-auto sm:w-full sm:max-w-md mb-6">
				<h2
					class="text-center text-3xl font-extrabold text-gray-900"
					data-testid="verify-email-heading"
				>
					メールアドレスの変更確認
				</h2>
			</div>

			{#if data.verificationFailed}
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

					<Button
						variant="primary"
						onclick={goToProfile}
						class="w-full"
						data-testid="back-to-profile-button"
					>
						プロフィール設定に戻る
					</Button>
				</div>
			{/if}
		</div>
	</div>
</div>

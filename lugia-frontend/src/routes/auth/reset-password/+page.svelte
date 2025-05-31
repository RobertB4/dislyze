<script lang="ts">
	import { toast } from "$components/Toast/toast";
	import { createForm } from "felte";
	import Button from "$components/Button.svelte";
	import Input from "$components/Input.svelte";
	import type { PageData } from "./$types";
	import { safeGoto } from "$lib/routing";

	export let data: PageData;

	const {
		form,
		errors,
		data: formData,
		isSubmitting
	} = createForm({
		initialValues: {
			password: "",
			password_confirm: ""
		},
		validate: (values) => {
			const errs: Record<string, string> = {};
			values.password = values.password.trim();
			values.password_confirm = values.password_confirm.trim();

			if (!values.password) {
				errs.password = "パスワードは必須です";
			} else if (values.password.length < 8) {
				errs.password = "パスワードは8文字以上である必要があります";
			}
			if (!values.password_confirm) {
				errs.password_confirm = "パスワード確認は必須です";
			} else if (values.password !== values.password_confirm) {
				errs.password_confirm = "パスワードが一致しません";
			}
			return errs;
		},
		onSubmit: async (values) => {
			try {
				const response = await fetch(`/api/auth/reset-password`, {
					method: "POST",
					headers: {
						"Content-Type": "application/json"
					},
					body: JSON.stringify({
						token: data.token,
						password: values.password,
						password_confirm: values.password_confirm
					})
				});

				if (response.ok) {
					safeGoto("/auth/login");
				} else {
					// API returned non-200 (e.g. 400 if token became invalid after page load, or 500)
					toast.show(
						"パスワードのリセットに失敗しました。もう一度お試しいただくか、再度リセットをリクエストしてください。",
						"error"
					);
				}
			} catch (err) {
				console.error("Error resetting password:", err);
				toast.show(
					"パスワードのリセット中にエラーが発生しました。ネットワーク接続を確認してください。",
					"error"
				);
			}
		}
	});
</script>

<main class="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
	<div class="max-w-md w-full space-y-8">
		<div>
			<a href="/">
				<img src="/logofull.png" alt="Dislyze Logo" class="mx-auto h-12 w-auto" />
			</a>
			<h2
				data-testid="reset-password-heading"
				class="mt-6 text-center text-3xl font-extrabold text-gray-900"
			>
				新しいパスワードを設定
			</h2>
		</div>

		{#if data.email && data.token}
			<form class="mt-8 space-y-6" use:form>
				<div class="rounded-md shadow-sm space-y-4">
					<Input
						id="email"
						name="email"
						type="email"
						label="メールアドレス"
						value={data.email}
						disabled
					/>
					<Input
						id="password"
						name="password"
						type="password"
						label="新しいパスワード"
						placeholder="新しいパスワード"
						required
						bind:value={$formData.password}
						error={$errors.password?.[0]}
					/>
					<Input
						id="password_confirm"
						name="password_confirm"
						type="password"
						label="新しいパスワード (確認)"
						placeholder="新しいパスワード (確認)"
						required
						bind:value={$formData.password_confirm}
						error={$errors.password_confirm?.[0]}
					/>
				</div>

				<div>
					<Button
						data-testid="reset-password-submit-button"
						type="submit"
						loading={$isSubmitting}
						fullWidth>パスワードをリセット</Button
					>
				</div>
			</form>
		{:else}
			<!-- This part should ideally not be reached if the load function correctly throws errors -->
			<div class="text-center py-4">
				<p data-testid="reset-password-token-error-message" class="text-red-600">
					リセットトークンの読み込み中にエラーが発生しました。リンクが正しいか確認してください。
				</p>
				<a
					data-testid="reset-password-retry-link"
					href="/auth/forgot-password"
					class="mt-4 font-medium text-indigo-600 hover:text-indigo-500"
				>
					パスワードリセットを再試行
				</a>
			</div>
		{/if}
		<div class="text-sm text-center">
			<a
				data-testid="reset-password-back-to-login-link"
				href="/auth/login"
				class="font-medium text-indigo-600 hover:text-indigo-500"
			>
				ログインページに戻る
			</a>
		</div>
	</div>
</main>

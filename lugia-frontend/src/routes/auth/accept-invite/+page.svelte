<script lang="ts">
	import { Button, toast, Input } from "@dislyze/zoroark";
	import { createForm } from "felte";
	import type { PageData } from "./$types";
	import { safeGoto } from "$lib/routing";
	import { mutationFetch } from "$lib/fetch";

	let { data: pageData }: { data: PageData } = $props();

	const showForm = pageData.token && pageData.inviterName && pageData.invitedEmail;

	const { form, data, errors, isValid, isSubmitting } = createForm({
		initialValues: {
			email: pageData.invitedEmail || "",
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
				errs.password_confirm = "パスワードを確認してください";
			} else if (values.password !== values.password_confirm) {
				errs.password_confirm = "パスワードが一致しません";
			}
			return errs;
		},
		onSubmit: async (values) => {
			const { success } = await mutationFetch(`/api/auth/accept-invite`, {
				method: "POST",
				headers: {
					"Content-Type": "application/json"
				},
				body: JSON.stringify({
					token: pageData.token,
					password: values.password,
					password_confirm: values.password_confirm
				})
			});

			if (success) {
				toast.show("招待が承認されました。", "success");
				safeGoto("/");
			}
		}
	});
</script>

<main class="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
	<div class="max-w-md w-full space-y-8">
		<div>
			<img class="mx-auto h-12 w-auto" src="/logo.png" alt="Your Company" />
		</div>

		{#if showForm}
			<div>
				<h2
					class="mt-6 text-center text-2xl font-extrabold text-gray-900"
					data-testid="accept-invite-title"
				>
					招待の承認
				</h2>
				{#if pageData.inviterName}
					<p class="mt-2 text-center text-sm text-gray-600" data-testid="inviter-message">
						{pageData.inviterName}さんがあなたを招待しました。
						<br />
						アカウントのパスワードを設定してください。
					</p>
				{/if}
			</div>

			<form class="mt-8 space-y-6" use:form data-testid="accept-invite-form">
				<div class="rounded-md space-y-4">
					<Input
						id="email"
						name="email"
						type="email"
						label="メールアドレス"
						bind:value={$data.email}
						disabled
					/>
					<Input
						id="password"
						name="password"
						type="password"
						label="新しいパスワード"
						placeholder="新しいパスワード"
						required
						bind:value={$data.password}
						error={$errors.password?.[0]}
					/>
					<Input
						id="password_confirm"
						name="password_confirm"
						type="password"
						label="新しいパスワード（確認）"
						placeholder="新しいパスワード（確認）"
						required
						bind:value={$data.password_confirm}
						error={$errors.password_confirm?.[0]}
					/>
				</div>

				<div>
					<Button
						type="submit"
						disabled={!$isValid}
						loading={$isSubmitting}
						fullWidth
						data-testid="submit-button"
					>
						招待を承認する
					</Button>
				</div>
			</form>
		{:else}
			<div class="text-center" data-testid="error-state">
				<h2 class="mt-6 text-2xl font-bold text-red-600" data-testid="error-title">エラー</h2>
				<p class="mt-2 text-lg text-red-500" data-testid="error-message">
					招待リンクが無効か、期限切れです。
					<br />
					お手数ですが、招待者に再度依頼してください。
				</p>
				<div class="mt-6">
					<a
						href="/auth/login"
						class="text-indigo-600 hover:text-indigo-500 font-medium"
						data-testid="login-link"
					>
						ログインページへ戻る
					</a>
				</div>
			</div>
		{/if}
	</div>
</main>

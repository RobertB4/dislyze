<script lang="ts">
	import { toast, Button, Input } from "@dislyze/zoroark";
	import { createForm } from "felte";

	const { form, errors, data, isSubmitting } = createForm({
		initialValues: {
			email: ""
		},
		validate: (values) => {
			const errs: Record<string, string> = {};
			values.email = values.email.trim();

			if (!values.email) {
				errs.email = "メールアドレスは必須です";
			} else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(values.email)) {
				errs.email = "メールアドレスの形式が正しくありません";
			}
			return errs;
		},
		onSubmit: async (values) => {
			try {
				const response = await fetch(`/api/auth/forgot-password`, {
					method: "POST",
					headers: {
						"Content-Type": "application/json"
					},
					body: JSON.stringify(values)
				});

				if (response.ok) {
					toast.show("パスワードリセットの手順を記載したメールを送信しました。", "success");
				} else {
					throw new Error(`/auth/forgot-password returned non-200 status code: ${response.status}`);
				}
			} catch (err) {
				toast.showError(err);
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
				data-testid="forgot-password-heading"
				class="mt-6 text-center text-3xl font-extrabold text-gray-900"
			>
				パスワードをお忘れですか？
			</h2>
			<p class="mt-2 text-center text-sm text-gray-600">
				メールアドレスを入力してください。パスワードリセットの手順をお送りします。
			</p>
		</div>

		<form class="mt-8 space-y-6" use:form>
			<div class="rounded-md">
				<Input
					id="email"
					name="email"
					type="email"
					label="メールアドレス"
					placeholder="メールアドレス"
					required
					bind:value={$data.email}
					error={$errors.email?.[0]}
				/>
			</div>

			<div>
				<Button
					data-testid="forgot-password-submit-button"
					type="submit"
					loading={$isSubmitting}
					fullWidth
				>
					パスワードリセットをリクエスト
				</Button>
			</div>
		</form>
		<div class="text-sm text-center">
			<a
				data-testid="back-to-login-link"
				href="/auth/login"
				class="font-medium text-indigo-600 hover:text-indigo-500"
			>
				ログインページに戻る
			</a>
		</div>
	</div>
</main>

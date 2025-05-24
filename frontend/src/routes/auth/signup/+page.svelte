<script lang="ts">
	import { toast } from "$components/Toast/toast";
	import { createForm } from "felte";
	import { PUBLIC_API_URL } from "$env/static/public";
	import Button from "$components/Button.svelte";
	import Input from "$components/Input.svelte";
	import { KnownError } from "$lib/errors";
	import { safeGoto } from "$lib/routing";

	const { form, errors, data, isSubmitting } = createForm({
		initialValues: {
			company_name: "",
			user_name: "",
			email: "",
			password: "",
			password_confirm: ""
		},
		validate: (values) => {
			const errors: Record<string, string> = {};

			values.company_name = values.company_name.trim();
			values.user_name = values.user_name.trim();
			values.email = values.email.trim();
			values.password = values.password.trim();
			values.password_confirm = values.password_confirm.trim();

			if (!values.company_name) {
				errors.company_name = "会社名は必須です";
			}
			if (!values.user_name) {
				errors.user_name = "氏名は必須です";
			}
			if (!values.email) {
				errors.email = "メールアドレスは必須です";
			} else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(values.email)) {
				errors.email = "メールアドレスの形式が正しくありません";
			}
			if (!values.password) {
				errors.password = "パスワードは必須です";
			} else if (values.password.length < 8) {
				errors.password = "パスワードは8文字以上である必要があります";
			}
			if (!values.password_confirm) {
				errors.password_confirm = "パスワードを確認してください";
			} else if (values.password !== values.password_confirm) {
				errors.password_confirm = "パスワードが一致しません";
			}

			return errors;
		},
		onSubmit: async (values) => {
			try {
				const response = await fetch(`${PUBLIC_API_URL}/auth/signup`, {
					method: "POST",
					headers: {
						"Content-Type": "application/json"
					},
					body: JSON.stringify(values),
					credentials: "include"
				});

				const data = await response.json();

				if (data.error) {
					throw new KnownError(data.error);
				}

				safeGoto("/");
			} catch (err) {
				toast.showError(err);
			}
		}
	});
</script>

<main class="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
	<div class="max-w-md w-full space-y-8">
		<div>
			<img src="/logofull.png" alt="Dislyze Logo" class="mx-auto h-12 w-auto" />
			<h2 class="mt-6 text-center text-3xl font-extrabold text-gray-900">アカウントを作成</h2>
			<p class="mt-2 text-center text-sm text-gray-600">
				または
				<a href="/auth/login" class="font-medium text-indigo-600 hover:text-indigo-500">
					既存のアカウントにログイン
				</a>
			</p>
		</div>

		<form class="mt-8 space-y-6" use:form>
			<div class="rounded-md space-y-4">
				<Input
					id="company_name"
					name="company_name"
					type="text"
					label="会社名"
					placeholder="会社名"
					required
					bind:value={$data.company_name}
					error={$errors.company_name?.[0]}
				/>

				<Input
					id="user_name"
					name="user_name"
					type="text"
					label="氏名"
					placeholder="氏名"
					required
					bind:value={$data.user_name}
					error={$errors.user_name?.[0]}
				/>

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

				<Input
					id="password"
					name="password"
					type="password"
					label="パスワード"
					placeholder="パスワード"
					required
					bind:value={$data.password}
					error={$errors.password?.[0]}
				/>

				<Input
					id="password_confirm"
					name="password_confirm"
					type="password"
					label="パスワード（確認）"
					placeholder="パスワード（確認）"
					required
					bind:value={$data.password_confirm}
					error={$errors.password_confirm?.[0]}
				/>
			</div>

			<div>
				<Button type="submit" loading={$isSubmitting} fullWidth>アカウントを作成</Button>
			</div>
		</form>
	</div>
</main>

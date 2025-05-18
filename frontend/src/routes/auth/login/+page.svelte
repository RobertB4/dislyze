<script lang="ts">
	import { goto } from "$app/navigation";
	import { toast } from "$components/Toast/toast";
	import { createForm } from "felte";
	import { PUBLIC_API_URL } from "$env/static/public";
	import Button from "$components/Button.svelte";
	import Input from "$components/Input.svelte";
	import { KnownError } from "$lib/errors";

	const { form, errors, data, isValid, isSubmitting } = createForm({
		initialValues: {
			email: "",
			password: ""
		},
		validate: (values) => {
			const errors: Record<string, string> = {};

			values.email = values.email.trim();
			values.password = values.password.trim();

			if (!values.email) {
				errors.email = "メールアドレスは必須です";
			} else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(values.email)) {
				errors.email = "メールアドレスの形式が正しくありません";
			}
			if (!values.password) {
				errors.password = "パスワードは必須です";
			}

			return errors;
		},
		onSubmit: async (values) => {
			try {
				const response = await fetch(`${PUBLIC_API_URL}/auth/login`, {
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
				goto("/");
			} catch (err) {
				toast.showError(err);
			}
		}
	});
</script>

<main class="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
	<div class="max-w-md w-full space-y-8">
		<div>
			<h2 class="mt-6 text-center text-3xl font-extrabold text-gray-900">ログイン</h2>
			<p class="mt-2 text-center text-sm text-gray-600">
				または
				<a href="/auth/signup" class="font-medium text-indigo-600 hover:text-indigo-500">
					新規アカウントを作成
				</a>
			</p>
		</div>

		<form class="mt-8 space-y-6" use:form>
			<div class="rounded-md space-y-4">
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
			</div>

			<div>
				<Button type="submit" disabled={!$isValid} loading={$isSubmitting} fullWidth>
					ログイン
				</Button>
			</div>
		</form>
	</div>
</main>

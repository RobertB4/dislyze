<script lang="ts">
	import { goto } from '$app/navigation';
	import { toast } from '$components/toast';
	import { createForm } from 'felte';
	import { PUBLIC_API_URL } from '$env/static/public';
	import Button from '$lib/components/Button.svelte';

	const { form, errors, data, isValid, isSubmitting } = createForm({
		initialValues: {
			company_name: '',
			user_name: '',
			email: '',
			password: '',
			password_confirm: ''
		},
		validate: (values) => {
			const errors: Record<string, string> = {};

			// Trim whitespace from all fields
			values.company_name = values.company_name.trim();
			values.user_name = values.user_name.trim();
			values.email = values.email.trim();
			values.password = values.password.trim();
			values.password_confirm = values.password_confirm.trim();

			// Check for empty or whitespace-only fields
			if (!values.company_name) {
				errors.company_name = '会社名は必須です';
			}
			if (!values.user_name) {
				errors.user_name = 'ユーザー名は必須です';
			}
			if (!values.email) {
				errors.email = 'メールアドレスは必須です';
			} else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(values.email)) {
				errors.email = 'メールアドレスの形式が正しくありません';
			}
			if (!values.password) {
				errors.password = 'パスワードは必須です';
			} else if (values.password.length < 8) {
				errors.password = 'パスワードは8文字以上である必要があります';
			}
			if (!values.password_confirm) {
				errors.password_confirm = 'パスワードを確認してください';
			} else if (values.password !== values.password_confirm) {
				errors.password_confirm = 'パスワードが一致しません';
			}

			return errors;
		},
		onSubmit: async (values) => {
			try {
				const response = await fetch(`${PUBLIC_API_URL}/auth/signup`, {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json'
					},
					body: JSON.stringify(values),
					credentials: 'include'
				});

				const data = await response.json();
				console.log({ data });

				if (data.error) {
					throw new Error(data.error);
				}

				// Show success toast and redirect to dashboard
				toast.show('アカウントが正常に作成されました！', 'success');
				goto('/dashboard');
			} catch (err) {
				console.log({ err });
				const errorMessage = err instanceof Error ? err.message : 'An error occurred';

				toast.show(errorMessage, 'error');
			}
		}
	});
</script>

<main class="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
	<div class="max-w-md w-full space-y-8">
		<div>
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
				<div>
					<label for="company_name" class="sr-only">会社名</label>
					<input
						id="company_name"
						name="company_name"
						type="text"
						required
						bind:value={$data.company_name}
						class="appearance-none rounded-md relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
						placeholder="会社名"
					/>
					{#if $errors.company_name}
						<p class="mt-1 text-sm text-red-600">{$errors.company_name}</p>
					{/if}
				</div>

				<div>
					<label for="user_name" class="sr-only">お名前</label>
					<input
						id="user_name"
						name="user_name"
						type="text"
						required
						bind:value={$data.user_name}
						class="appearance-none rounded-md relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
						placeholder="お名前"
					/>
					{#if $errors.user_name}
						<p class="mt-1 text-sm text-red-600">{$errors.user_name}</p>
					{/if}
				</div>

				<div>
					<label for="email" class="sr-only">メールアドレス</label>
					<input
						id="email"
						name="email"
						type="email"
						required
						bind:value={$data.email}
						class="appearance-none rounded-md relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
						placeholder="メールアドレス"
					/>
					{#if $errors.email}
						<p class="mt-1 text-sm text-red-600">{$errors.email}</p>
					{/if}
				</div>

				<div>
					<label for="password" class="sr-only">パスワード</label>
					<input
						id="password"
						name="password"
						type="password"
						required
						bind:value={$data.password}
						class="appearance-none rounded-md relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
						placeholder="パスワード"
					/>
					{#if $errors.password}
						<p class="mt-1 text-sm text-red-600">{$errors.password}</p>
					{/if}
				</div>

				<div>
					<label for="password_confirm" class="sr-only">パスワード（確認）</label>
					<input
						id="password_confirm"
						name="password_confirm"
						type="password"
						required
						bind:value={$data.password_confirm}
						class="appearance-none rounded-md relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
						placeholder="パスワード（確認）"
					/>
					{#if $errors.password_confirm}
						<p class="mt-1 text-sm text-red-600">{$errors.password_confirm}</p>
					{/if}
				</div>
			</div>

			<div>
				<Button type="submit" disabled={!$isValid} loading={$isSubmitting} fullWidth>
					アカウントを作成
				</Button>
			</div>
		</form>
	</div>
</main>

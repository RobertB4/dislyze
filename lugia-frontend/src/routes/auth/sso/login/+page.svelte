<script lang="ts">
	import { toast, Button, Input, Alert, KnownError } from "@dislyze/zoroark";
	import { createForm } from "felte";
	import type { PageData } from "./$types";
	import { resolve } from "$app/paths";

	let { data: pageData }: { data: PageData } = $props();

	const { form, errors, data, isSubmitting } = createForm({
		initialValues: {
			email: ""
		},
		validate: (values) => {
			const errors: Record<string, string> = {};

			values.email = values.email.trim();

			if (!values.email) {
				errors.email = "メールアドレスは必須です";
			} else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(values.email)) {
				errors.email = "メールアドレスの形式が正しくありません";
			}

			return errors;
		},
		onSubmit: async (values) => {
			try {
				const response = await fetch(`/api/auth/sso/login`, {
					method: "POST",
					headers: {
						"Content-Type": "application/json"
					},
					body: JSON.stringify(values)
				});

				if (!response.ok) {
					throw new KnownError("SSOログインに失敗しました。");
				}

				const { html } = (await response.json()) as { html: string };

				const container = document.createElement("div");
				container.innerHTML = html;
				document.body.appendChild(container);

				const form = container.querySelector("form");
				if (!form) {
					throw new Error("expected form");
				}
				form.submit();
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
			<h2
				data-testid="sso-login-heading"
				class="mt-6 text-center text-3xl font-extrabold text-gray-900"
			>
				SSOでログイン
			</h2>
		</div>

		{#if pageData.error}
			<Alert type="danger" data-testid="sso-login-error">
				<p data-testid="sso-login-error-message" class="text-sm">{pageData.error}</p>
			</Alert>
		{/if}

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
			</div>

			<div>
				<Button
					data-testid="sso-login-submit-button"
					type="submit"
					loading={$isSubmitting}
					fullWidth>SSOでログイン</Button
				>
			</div>

			<div class="flex items-center justify-center">
				<div class="text-sm">
					<a
						data-testid="regular-login-link"
						href={resolve("/auth/login")}
						class="font-medium text-indigo-600 hover:text-indigo-500"
					>
						パスワードでログインする方はこちら
					</a>
				</div>
			</div>
		</form>
	</div>
</main>

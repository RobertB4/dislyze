<script lang="ts">
	import { Button, toast, Input, KnownError } from "@dislyze/zoroark";
	import { createForm } from "felte";
	import type { PageData } from "./$types";
	import { safeGoto } from "@dislyze/zoroark";
	import { mutationFetch } from "$lib/fetch";
	import { resolve } from "$app/paths";

	let { data: pageData }: { data: PageData } = $props();

	const showForm = pageData.token && pageData.email;

	const { form, data, errors, isSubmitting } = createForm({
		initialValues: {
			email: pageData.email || "",
			company_name: pageData.companyName || "",
			user_name: pageData.userName || "",
			password: "",
			password_confirm: ""
		},
		validate: (values) => {
			const errs: Record<string, string> = {};
			values.company_name = values.company_name.trim();
			values.user_name = values.user_name.trim();

			if (!values.company_name) {
				errs.company_name = "会社名は必須です";
			}

			if (!values.user_name) {
				errs.user_name = "氏名は必須です";
			}

			if (!pageData.ssoEnabled) {
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
			}

			return errs;
		},
		onSubmit: async (values) => {
			try {
				const encodedToken = encodeURIComponent(pageData.token);
				const { success } = await mutationFetch(`/api/auth/tenant-signup?token=${encodedToken}`, {
					method: "POST",
					headers: {
						"Content-Type": "application/json"
					},
					body: JSON.stringify({
						password: values.password,
						password_confirm: values.password_confirm,
						company_name: values.company_name,
						user_name: values.user_name
					})
				});

				if (success) {
					if (pageData.ssoEnabled) {
						const response = await fetch(`/api/auth/sso/login`, {
							method: "POST",
							headers: {
								"Content-Type": "application/json"
							},
							body: JSON.stringify({
								email: pageData.email
							})
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
					} else {
						toast.show("アカウントが作成されました。", "success");
						safeGoto("/");
					}
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
			<img class="mx-auto h-12 w-auto" src="/logofull.png" alt="Dislyze" />
		</div>

		{#if showForm}
			<div>
				<h2
					class="mt-6 text-center text-2xl font-extrabold text-gray-900"
					data-testid="signup-title"
				>
					アカウントを作成
				</h2>
			</div>

			<form class="mt-8 space-y-6" use:form data-testid="signup-form">
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
						id="company_name"
						name="company_name"
						type="text"
						label="会社名"
						placeholder="会社名"
						required
						bind:value={$data.company_name}
						error={$errors.company_name?.[0]}
					/>
					{#if !pageData.ssoEnabled}
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
					{/if}
				</div>

				<div>
					<Button type="submit" loading={$isSubmitting} fullWidth data-testid="signup-button">
						アカウントを作成
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
						href={resolve("/auth/login")}
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

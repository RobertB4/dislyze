<script lang="ts">
	import Layout from "$components/Layout.svelte";
	import Button from "$components/Button.svelte";
	import Input from "$components/Input.svelte";
	import type { PageData } from "./$types";
	import { createForm } from "felte";
	import { toast } from "$components/Toast/toast";
	import { mutationFetch } from "$lib/fetch";
	import { forceUpdateMeCache } from "$lib/meCache";
	import { invalidate } from "$app/navigation";

	let { data: pageData }: { data: PageData } = $props();

	const {
		form: nameForm,
		data: nameData,
		errors: nameErrors,
		isSubmitting: nameSubmitting
	} = createForm({
		initialValues: {
			name: pageData.me.user_name
		},
		validate: (values) => {
			const errs: Record<string, string> = {};
			values.name = values.name.trim();

			if (!values.name) {
				errs.name = "氏名は必須です";
			}
			return errs;
		},
		onSubmit: async (values) => {
			const { success } = await mutationFetch(`/api/me/change-name`, {
				method: "POST",
				headers: {
					"Content-Type": "application/json"
				},
				body: JSON.stringify({ name: values.name })
			});

			if (success) {
				// Force the layout to refresh meCache with fresh data
				// Needed to ensure the updated name is reflected in the UI
				// see +layout.ts for more details
				forceUpdateMeCache.set(true);
				await invalidate((u) => u.pathname === "/api/me");
				toast.show("氏名を更新しました。", "success");
			}
		}
	});
</script>

<Layout me={pageData.me} pageTitle="プロフィール設定">
	<div class="max-w-2xl mx-auto space-y-8">
		<!-- Change Name Section -->
		<div class="bg-white shadow rounded-lg p-6" data-testid="change-name-section">
			<h2 class="text-lg font-medium text-gray-900 mb-4">氏名を変更</h2>
			<form use:nameForm class="space-y-4" data-testid="change-name-form">
				<Input
					id="name"
					name="name"
					type="text"
					variant="underlined"
					label="氏名"
					placeholder="氏名を入力してください"
					bind:value={$nameData.name}
					error={$nameErrors.name?.[0]}
					required
				/>
				<Button
					type="submit"
					variant="primary"
					loading={$nameSubmitting}
					disabled={$nameSubmitting}
					data-testid="save-name-button"
				>
					氏名を保存
				</Button>
			</form>
		</div>

		<!-- Change Password Section -->
		<div class="bg-white shadow rounded-lg p-6" data-testid="change-password-section">
			<h2 class="text-lg font-medium text-gray-900 mb-4">パスワードを変更</h2>
			<form class="space-y-4" data-testid="change-password-form">
				<Input
					id="current-password"
					name="current-password"
					type="password"
					variant="underlined"
					label="現在のパスワード"
					placeholder="現在のパスワードを入力してください"
					required
				/>
				<Input
					id="new-password"
					name="new-password"
					type="password"
					variant="underlined"
					label="新しいパスワード"
					placeholder="新しいパスワードを入力してください"
					required
				/>
				<Input
					id="confirm-password"
					name="confirm-password"
					type="password"
					variant="underlined"
					label="新しいパスワード（確認）"
					placeholder="新しいパスワードを再度入力してください"
					required
				/>
				<Button type="submit" variant="primary" data-testid="save-password-button">
					パスワードを保存
				</Button>
			</form>
		</div>

		<!-- Change Email Section -->
		<div class="bg-white shadow rounded-lg p-6" data-testid="change-email-section">
			<h2 class="text-lg font-medium text-gray-900 mb-4">メールアドレスを変更</h2>
			<form class="space-y-4" data-testid="change-email-form">
				<div>
					<label class="block text-sm font-medium text-gray-700 mb-1">現在のメールアドレス</label>
					<p class="text-sm text-gray-600" data-testid="current-email">{pageData.me.email}</p>
				</div>
				<Input
					id="new-email"
					name="new-email"
					type="email"
					variant="underlined"
					label="新しいメールアドレス"
					placeholder="新しいメールアドレスを入力してください"
					required
				/>
				<Button type="submit" variant="primary" data-testid="save-email-button">
					メールアドレスを保存
				</Button>
			</form>
		</div>

		<!-- Change Tenant Name Section (Admin Only) -->
		{#if pageData.me.user_role === "admin"}
			<div class="bg-white shadow rounded-lg p-6" data-testid="change-tenant-section">
				<h2 class="text-lg font-medium text-gray-900 mb-4">組織名を変更</h2>
				<form class="space-y-4" data-testid="change-tenant-form">
					<Input
						id="tenant-name"
						name="tenant-name"
						type="text"
						variant="underlined"
						label="組織名"
						placeholder="組織名を入力してください"
						value={pageData.me.tenant_name}
						required
					/>
					<Button type="submit" variant="primary" data-testid="save-tenant-button">
						組織名を保存
					</Button>
				</form>
			</div>
		{/if}
	</div>
</Layout>

<script lang="ts">
	import Layout from "$components/Layout.svelte";
	import Button from "$components/Button.svelte";
	import Input from "$components/Input.svelte";
	import SettingsTabs from "../SettingsTabs.svelte";
	import type { PageData } from "./$types";
	import { createForm } from "felte";
	import { toast } from "$components/Toast/toast";
	import { mutationFetch } from "$lib/fetch";
	import { forceUpdateMeCache, hasPermission } from "$lib/meCache";
	import { invalidate } from "$app/navigation";
	import { page } from "$app/state";

	let { data: pageData }: { data: PageData } = $props();

	if (page.url.searchParams.get("email-verified") === "true") {
		toast.show("メールアドレスの変更が完了しました。", "success");
		// Force refresh to get updated email
		forceUpdateMeCache.set(true);
		invalidate((u) => u.pathname === "/api/me");
	}

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

	const {
		form: passwordForm,
		data: passwordData,
		errors: passwordErrors,
		isSubmitting: passwordSubmitting,
		reset: passwordReset
	} = createForm({
		initialValues: {
			currentPassword: "",
			newPassword: "",
			confirmPassword: ""
		},
		validate: (values) => {
			const errs: Record<string, string> = {};
			values.currentPassword = values.currentPassword.trim();
			values.newPassword = values.newPassword.trim();
			values.confirmPassword = values.confirmPassword.trim();

			if (!values.currentPassword) {
				errs.currentPassword = "現在のパスワードは必須です";
			}

			if (!values.newPassword) {
				errs.newPassword = "新しいパスワードは必須です";
			} else if (values.newPassword.length < 8) {
				errs.newPassword = "パスワードは8文字以上である必要があります";
			}

			if (!values.confirmPassword) {
				errs.confirmPassword = "新しいパスワード（確認）は必須です";
			} else if (values.newPassword !== values.confirmPassword) {
				errs.confirmPassword = "パスワードが一致しません";
			}

			return errs;
		},
		onSubmit: async (values) => {
			const { success } = await mutationFetch(`/api/me/change-password`, {
				method: "POST",
				headers: {
					"Content-Type": "application/json"
				},
				body: JSON.stringify({
					current_password: values.currentPassword,
					new_password: values.newPassword,
					new_password_confirm: values.confirmPassword
				})
			});

			if (success) {
				passwordReset();
				toast.show("パスワードを更新しました。", "success");
			}
		}
	});

	const {
		form: emailForm,
		data: emailData,
		errors: emailErrors,
		isSubmitting: emailSubmitting,
		reset: emailReset
	} = createForm({
		initialValues: {
			newEmail: ""
		},
		validate: (values) => {
			const errs: Record<string, string> = {};
			values.newEmail = values.newEmail.trim();

			if (!values.newEmail) {
				errs.newEmail = "新しいメールアドレスは必須です";
			} else if (!values.newEmail.includes("@")) {
				errs.newEmail = "有効なメールアドレスを入力してください";
			} else if (values.newEmail === pageData.me.email) {
				errs.newEmail = "現在のメールアドレスと同じです";
			}

			return errs;
		},
		onSubmit: async (values) => {
			const { success } = await mutationFetch(`/api/me/change-email`, {
				method: "POST",
				headers: {
					"Content-Type": "application/json"
				},
				body: JSON.stringify({
					new_email: values.newEmail
				})
			});

			if (success) {
				emailReset();
				toast.show("確認メールを送信しました。メールをご確認ください。", "success");
			}
		}
	});

	const {
		form: tenantForm,
		data: tenantData,
		errors: tenantErrors,
		isSubmitting: tenantSubmitting
	} = createForm({
		initialValues: {
			tenantName: pageData.me.tenant_name
		},
		validate: (values) => {
			const errs: Record<string, string> = {};
			values.tenantName = values.tenantName.trim();

			if (!values.tenantName) {
				errs.tenantName = "組織名は必須です";
			}

			return errs;
		},
		onSubmit: async (values) => {
			const { success } = await mutationFetch(`/api/tenant/change-name`, {
				method: "POST",
				headers: {
					"Content-Type": "application/json"
				},
				body: JSON.stringify({
					name: values.tenantName
				})
			});

			if (success) {
				// Force the layout to refresh meCache with fresh data
				// Needed to ensure the updated tenant name is reflected in the UI
				forceUpdateMeCache.set(true);
				await invalidate((u) => u.pathname === "/api/me");
				toast.show("組織名を更新しました。", "success");
			}
		}
	});

	const navigationItems = [
		{ id: "change-name-section", label: "氏名を変更" },
		{ id: "change-password-section", label: "パスワードを変更" },
		{ id: "change-email-section", label: "メールアドレスを変更" },
		...(hasPermission(pageData.me, "tenant.update")
			? [{ id: "change-tenant-section", label: "組織名を変更" }]
			: [])
	];

	function scrollToSection(sectionId: string) {
		const element = document.getElementById(sectionId);
		if (element) {
			const elementPosition = element.getBoundingClientRect().top + window.pageYOffset;
			const offsetPosition = elementPosition - 62; // Account for sticky header height

			window.scrollTo({
				top: offsetPosition,
				behavior: "smooth"
			});
		}
	}
</script>

<Layout me={pageData.me} pageTitle="プロフィール設定">
	<div>
		<SettingsTabs me={pageData.me} />

		<div class="flex gap-8">
			<!-- Left Navigation Menu (Hidden on small screens) -->
			<div class="hidden lg:block w-64 flex-shrink-0">
				<div
					class="sticky top-24 bg-white shadow rounded-lg border border-gray-200 p-4"
					data-testid="profile-navigation"
				>
					<nav class="space-y-1">
						{#each navigationItems as item (item.id)}
							<button
								type="button"
								onclick={() => scrollToSection(item.id)}
								class="w-full text-left px-3 py-2 text-sm text-gray-600 cursor-pointer hover:text-gray-900 hover:bg-gray-50 rounded-md transition-colors duration-200"
								data-testid={`nav-${item.id}`}
							>
								{item.label}
							</button>
						{/each}
					</nav>
				</div>
			</div>

			<!-- Main Content -->
			<div class="flex-1 max-w-2xl space-y-8">
				<!-- Change Name Section -->
				<div
					id="change-name-section"
					class="bg-white shadow rounded-lg p-6"
					data-testid="change-name-section"
				>
					<h2 class="text-lg font-medium text-gray-900 mb-4" data-testid="change-name-heading">
						氏名を変更
					</h2>
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
				<div
					id="change-password-section"
					class="bg-white shadow rounded-lg p-6"
					data-testid="change-password-section"
				>
					<h2 class="text-lg font-medium text-gray-900 mb-4" data-testid="change-password-heading">
						パスワードを変更
					</h2>
					<form use:passwordForm class="space-y-4" data-testid="change-password-form">
						<Input
							id="currentPassword"
							name="currentPassword"
							type="password"
							variant="underlined"
							label="現在のパスワード"
							placeholder="現在のパスワードを入力してください"
							bind:value={$passwordData.currentPassword}
							error={$passwordErrors.currentPassword?.[0]}
							required
						/>
						<Input
							id="newPassword"
							name="newPassword"
							type="password"
							variant="underlined"
							label="新しいパスワード"
							placeholder="新しいパスワードを入力してください"
							bind:value={$passwordData.newPassword}
							error={$passwordErrors.newPassword?.[0]}
							required
						/>
						<Input
							id="confirmPassword"
							name="confirmPassword"
							type="password"
							variant="underlined"
							label="新しいパスワード（確認）"
							placeholder="新しいパスワードを再度入力してください"
							bind:value={$passwordData.confirmPassword}
							error={$passwordErrors.confirmPassword?.[0]}
							required
						/>
						<Button
							type="submit"
							variant="primary"
							loading={$passwordSubmitting}
							disabled={$passwordSubmitting}
							data-testid="save-password-button"
						>
							パスワードを保存
						</Button>
					</form>
				</div>

				<!-- Change Email Section -->
				<div
					id="change-email-section"
					class="bg-white shadow rounded-lg p-6"
					data-testid="change-email-section"
				>
					<h2 class="text-lg font-medium text-gray-900 mb-4" data-testid="change-email-heading">
						メールアドレスを変更
					</h2>
					<form use:emailForm class="space-y-4" data-testid="change-email-form">
						<div>
							<p class="block text-sm font-medium text-gray-700 mb-1">現在のメールアドレス</p>
							<p class="text-sm text-gray-600" data-testid="current-email">{pageData.me.email}</p>
						</div>
						<Input
							id="newEmail"
							name="newEmail"
							type="email"
							variant="underlined"
							label="新しいメールアドレス"
							placeholder="新しいメールアドレスを入力してください"
							bind:value={$emailData.newEmail}
							error={$emailErrors.newEmail?.[0]}
							required
						/>
						<Button
							type="submit"
							variant="primary"
							loading={$emailSubmitting}
							disabled={$emailSubmitting}
							data-testid="save-email-button"
						>
							確認メールを送信
						</Button>
					</form>
				</div>

				<!-- Change Tenant Name Section -->
				{#if hasPermission(pageData.me, "tenant.update")}
					<div
						id="change-tenant-section"
						class="bg-white shadow rounded-lg p-6"
						data-testid="change-tenant-section"
					>
						<h2 class="text-lg font-medium text-gray-900 mb-4" data-testid="change-tenant-heading">
							組織名を変更
						</h2>
						<form use:tenantForm class="space-y-4" data-testid="change-tenant-form">
							<Input
								id="tenantName"
								name="tenantName"
								type="text"
								variant="underlined"
								label="組織名"
								placeholder="組織名を入力してください"
								bind:value={$tenantData.tenantName}
								error={$tenantErrors.tenantName?.[0]}
								required
							/>
							<Button
								type="submit"
								variant="primary"
								loading={$tenantSubmitting}
								disabled={$tenantSubmitting}
								data-testid="save-tenant-button"
							>
								組織名を保存
							</Button>
						</form>
					</div>
				{/if}
			</div>
		</div>
	</div></Layout
>

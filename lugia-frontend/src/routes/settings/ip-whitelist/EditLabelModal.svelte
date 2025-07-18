<script lang="ts">
	import { Slideover, Input, toast } from "@dislyze/zoroark";
	import { createForm } from "felte";
	import { invalidate } from "$app/navigation";
	import { mutationFetch } from "$lib/fetch";
	import type { IPWhitelistRule } from "./+page";

	let {
		onClose,
		rule
	}: {
		onClose: () => void;
		rule: IPWhitelistRule;
	} = $props();

	const { form, data, errors, isSubmitting, reset } = createForm({
		initialValues: {
			label: rule.label || ""
		},
		validate: (values) => {
			const errs: Record<string, string> = {};

			values.label = values.label.trim();

			if (values.label && values.label.length > 255) {
				errs.label = "ラベルは255文字以内で入力してください";
			}

			return errs;
		},
		onSubmit: async (values) => {
			const payload = {
				label: values.label || null
			};

			const { success } = await mutationFetch(`/api/ip-whitelist/${rule.id}/label/update`, {
				method: "POST",
				headers: {
					"Content-Type": "application/json"
				},
				body: JSON.stringify(payload)
			});

			if (success) {
				toast.show("ラベルを更新しました", "success");
				handleClose();
				await invalidate((u) => u.pathname.includes("/api/ip-whitelist"));
			}
		}
	});

	const handleClose = () => {
		reset();
		onClose();
	};
</script>

<form use:form class="space-y-6 p-1 flex flex-col h-full" data-testid="edit-label-form">
	<Slideover
		title="ラベルを編集"
		subtitle={`${rule.ip_address} のラベルを編集`}
		primaryButtonText="更新"
		primaryButtonTypeSubmit={true}
		onClose={handleClose}
		loading={$isSubmitting}
		data-testid="edit-label-slideover"
	>
		<div class="flex-grow space-y-6">
			<div class="p-3 bg-gray-50 rounded-lg">
				<div class="text-sm text-gray-600 mb-1">IPアドレス</div>
				<code class="text-sm font-mono text-gray-900">{rule.ip_address}</code>
			</div>

			<Input
				id="label"
				name="label"
				type="text"
				label="ラベル"
				bind:value={$data.label}
				error={$errors.label?.[0]}
				placeholder="例: オフィスネットワーク（空欄にすると削除されます）"
				variant="underlined"
				data-testid="edit-label-input"
			/>

			<div class="text-sm text-gray-500">ラベルを空欄にすると削除されます。</div>
		</div>
	</Slideover>
</form>

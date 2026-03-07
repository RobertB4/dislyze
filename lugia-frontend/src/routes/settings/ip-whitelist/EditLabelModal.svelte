<script lang="ts">
	import Input from "@dislyze/zoroark/Input";
	import Slideover from "@dislyze/zoroark/Slideover";
	import { toast } from "@dislyze/zoroark/toast";
	import { createForm } from "felte";
	import { invalidate } from "$app/navigation";
	import { createMutationClient } from "$lugia/lib/api";
	import type { IpWhitelistRule } from "$lugia/schema";

	let {
		onClose,
		rule
	}: {
		onClose: () => void;
		rule: IpWhitelistRule;
	} = $props();

	const { form, data, errors, isSubmitting, reset } = createForm({
		initialValues: {
			label: rule.label || ""
		},
		validate: (values) => {
			const errs: Record<string, string> = {};

			values.label = values.label.trim();

			if (values.label && values.label.length > 255) {
				errs.label = "説明は255文字以内で入力してください";
			}

			return errs;
		},
		onSubmit: async (values) => {
			const api = createMutationClient();
			const { error } = await api.POST("/ip-whitelist/{id}/label/update", {
				params: { path: { id: rule.id } },
				body: { label: values.label || null }
			});

			if (!error) {
				toast.show("説明を更新しました", "success");
				handleClose();
				await invalidate((u) => u.pathname.includes("/api/ip-whitelist"));
			}
		}
	});

	function handleClose() {
		reset();
		onClose();
	}
</script>

<form use:form class="space-y-6 p-1 flex flex-col h-full" data-testid="edit-label-form">
	<Slideover
		title="説明を編集"
		subtitle={`${rule.ip_address} の説明を編集`}
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
				label="説明"
				bind:value={$data.label}
				error={$errors.label?.[0]}
				placeholder="例: オフィスネットワーク（空欄にすると削除されます）"
				variant="underlined"
				data-testid="edit-label-input"
			/>
		</div>
	</Slideover>
</form>

<script lang="ts">
	import { Slideover, Input, toast, Alert } from "@dislyze/zoroark";
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
			confirmIpAddress: ""
		},
		validate: (values) => {
			const errs: Record<string, string> = {};

			values.confirmIpAddress = values.confirmIpAddress.trim();

			if (!values.confirmIpAddress) {
				errs.confirmIpAddress = "IPアドレスの入力は必須です";
			} else if (values.confirmIpAddress !== rule.ip_address) {
				errs.confirmIpAddress = "IPアドレスが一致しません";
			}

			return errs;
		},
		onSubmit: async () => {
			const { success } = await mutationFetch(`/api/ip-whitelist/${rule.id}/delete`, {
				method: "POST"
			});

			if (success) {
				await invalidate((u) => u.pathname.includes("/api/ip-whitelist"));
				toast.show("IPアドレスを削除しました", "success");
				handleClose();
			}
		}
	});

	const handleClose = () => {
		reset();
		onClose();
	};
</script>

<form use:form class="space-y-6 p-1 flex flex-col h-full" data-testid="delete-ip-form">
	<Slideover
		title="IPアドレスを削除"
		subtitle="このIPアドレスを削除してもよろしいですか？"
		primaryButtonText="削除"
		primaryButtonTypeSubmit={true}
		onClose={handleClose}
		loading={$isSubmitting}
		data-testid="delete-ip-slideover"
	>
		<div class="flex-grow space-y-6">
			<Alert type="danger" title="この操作は元に戻せません" data-testid="delete-ip-warning">
				<p class="mb-3">
					IPアドレス <code class="bg-red-100 px-2 py-1 rounded text-red-800">{rule.ip_address}</code
					> を削除します。
				</p>
				<p>
					このIPアドレスからアクセスしているユーザーは、IPアドレス制限が有効な場合、アプリケーションにアクセスできなくなります。
				</p>
			</Alert>

			<Input
				id="confirmIpAddress"
				name="confirmIpAddress"
				type="text"
				label="確認のため、IPアドレスを入力してください"
				bind:value={$data.confirmIpAddress}
				error={$errors.confirmIpAddress?.[0]}
				required
				placeholder={rule.ip_address}
				variant="underlined"
				data-testid="confirm-ip-input"
			/>
		</div>
	</Slideover>
</form>

<script lang="ts">
	import Alert from "@dislyze/zoroark/Alert";
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
			const api = createMutationClient();
			const { error } = await api.POST("/ip-whitelist/{id}/delete", {
				params: { path: { id: rule.id } }
			});

			if (!error) {
				await invalidate((u) => u.pathname.includes("/api/ip-whitelist"));
				toast.show("IPアドレスを削除しました", "success");
				handleClose();
			}
		}
	});

	function handleClose() {
		reset();
		onClose();
	}
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

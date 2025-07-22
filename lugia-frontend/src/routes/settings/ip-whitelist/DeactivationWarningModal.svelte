<script lang="ts">
	import { Slideover, toast, Alert } from "@dislyze/zoroark";
	import { invalidate } from "$app/navigation";
	import { mutationFetch } from "$lib/fetch";
	import { forceUpdateMeCache } from "@dislyze/zoroark";

	let {
		onClose
	}: {
		onClose: () => void;
	} = $props();

	let isSubmitting = $state(false);

	const handleDeactivate = async () => {
		isSubmitting = true;

		const { success } = await mutationFetch(`/api/ip-whitelist/deactivate`, {
			method: "POST"
		});

		if (success) {
			forceUpdateMeCache.set(true);
			await invalidate((u) => u.pathname.includes("/api/me"));
			toast.show("IPアドレス制限を無効にしました", "success");
			onClose();
		}

		isSubmitting = false;
	};

	const handleClose = () => {
		onClose();
	};
</script>

<Slideover
	title="IPアドレス制限を無効にする"
	subtitle="確認してください"
	primaryButtonText="無効にする"
	onClose={handleClose}
	onPrimaryClick={handleDeactivate}
	loading={isSubmitting}
	data-testid="deactivation-warning-slideover"
>
	<div class="space-y-6">
		<Alert
			type="warning"
			title="セキュリティに関する重要な通知"
			data-testid="deactivation-warning-alert"
		>
			<div class="space-y-3">
				<p>
					IPアドレス制限を無効にすると、<strong>すべてのIPアドレス</strong
					>からアプリケーションにアクセスできるようになります。
				</p>
				<p class="font-medium">本当に無効にしますか？</p>
			</div>
		</Alert>

		<div class="p-4 bg-gray-50 border border-gray-200 rounded-lg">
			<h4 class="font-medium text-gray-900 mb-2">注意事項</h4>
			<p class="text-sm text-gray-700">
				IPアドレス制限を無効にした後でも、登録されているIPアドレスは保持されます。いつでも再度有効にすることができます。
			</p>
		</div>
	</div>
</Slideover>

<script lang="ts">
	import { Slideover, toast, Alert } from "@dislyze/zoroark";
	import { invalidate } from "$app/navigation";
	import { mutationFetch } from "$lib/fetch";
	import { forceUpdateMeCache } from "@dislyze/zoroark";

	let {
		onClose,
		userIP
	}: {
		onClose: () => void;
		userIP: string;
	} = $props();

	let isSubmitting = $state(false);

	const handleActivate = async () => {
		isSubmitting = true;

		const { success } = await mutationFetch(`/api/ip-whitelist/activate`, {
			method: "POST",
			headers: {
				"Content-Type": "application/json"
			},
			body: JSON.stringify({ force: true })
		});

		if (success) {
			forceUpdateMeCache.set(true);
			await invalidate((u) => u.pathname.includes("/api/ip-whitelist"));
			toast.show("IPアドレス制限を有効にしました", "success");
			onClose();
		}

		isSubmitting = false;
	};

	const handleClose = () => {
		onClose();
	};
</script>

<Slideover
	title="IPアドレス制限を有効にする"
	primaryButtonText="有効にする"
	onClose={handleClose}
	onPrimaryClick={handleActivate}
	loading={isSubmitting}
	data-testid="activation-warning-slideover"
>
	<div class="space-y-6">
		<Alert type="danger" title="アクセスできなくなります" data-testid="activation-warning-alert">
			<div class="space-y-3">
				<p>
					現在のIPアドレス: <code class="bg-red-100 px-2 py-1 rounded text-red-800">{userIP}</code>
				</p>
				<p>
					このIPアドレスはIP制限の対象外です。IPアドレス制限を有効にすると、このIPアドレスからアプリケーションにアクセスできなくなります。
				</p>
				<p class="font-medium">本当に続行しますか？</p>
			</div>
		</Alert>

		<div class="p-4 bg-blue-50 border border-blue-200 rounded-lg">
			<h4 class="font-medium text-blue-900 mb-2">緊急時の対応</h4>
			<p class="text-sm text-blue-800">
				アクセスできなくなった場合は、緊急解除用のメールをお送りします。
			</p>
		</div>
	</div>
</Slideover>

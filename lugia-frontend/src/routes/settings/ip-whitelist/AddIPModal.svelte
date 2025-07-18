<script lang="ts">
	import { Slideover, Input, toast } from "@dislyze/zoroark";
	import { createForm } from "felte";
	import { invalidate } from "$app/navigation";
	import { mutationFetch } from "$lib/fetch";
	import type { IPWhitelistRule } from "./+page";

	let {
		onClose,
		existingRules
	}: {
		onClose: () => void;
		existingRules: IPWhitelistRule[];
	} = $props();

	function validateIPOrCIDR(value: string): string | null {
		const trimmed = value.trim();
		if (!trimmed) return null;

		// Check if it's a single IP address
		if (!trimmed.includes("/")) {
			// Validate IPv4
			const ipv4Regex =
				/^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/;
			// Validate IPv6 (simplified)
			const ipv6Regex = /^(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$|^::1$|^::$/;

			if (ipv4Regex.test(trimmed) || ipv6Regex.test(trimmed)) {
				return null;
			}
			return "IPアドレスの形式が正しくありません";
		}

		// Check if it's a valid CIDR
		const cidrRegex =
			/^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\/(?:[0-9]|[1-2][0-9]|3[0-2])$|^(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\/(?:[0-9]|[1-9][0-9]|1[0-1][0-9]|12[0-8])$/;

		if (cidrRegex.test(trimmed)) {
			return null;
		}
		return "CIDR形式が正しくありません";
	}

	function isDuplicateIP(value: string): boolean {
		const trimmed = value.trim();
		return existingRules.some((rule) => rule.ip_address === trimmed);
	}

	const { form, data, errors, isSubmitting, reset } = createForm({
		initialValues: {
			ip_address: "",
			label: ""
		},
		validate: (values) => {
			const errs: Record<string, string> = {};

			values.ip_address = values.ip_address.trim();
			values.label = values.label.trim();

			if (!values.ip_address) {
				errs.ip_address = "IPアドレスは必須です";
			} else {
				const ipValidationError = validateIPOrCIDR(values.ip_address);
				if (ipValidationError) {
					errs.ip_address = ipValidationError;
				} else if (isDuplicateIP(values.ip_address)) {
					errs.ip_address = "このIPアドレスは既に登録されています";
				}
			}

			if (values.label && values.label.length > 255) {
				errs.label = "ラベルは255文字以内で入力してください";
			}

			return errs;
		},
		onSubmit: async (values) => {
			const payload = {
				ip_address: values.ip_address,
				label: values.label || null
			};

			const { success } = await mutationFetch(`/api/ip-whitelist/create`, {
				method: "POST",
				headers: {
					"Content-Type": "application/json"
				},
				body: JSON.stringify(payload)
			});

			if (success) {
				await invalidate((u) => u.pathname.includes("/api/ip-whitelist"));
				toast.show("IPアドレスを追加しました", "success");
				handleClose();
			}
		}
	});

	const handleClose = () => {
		reset();
		onClose();
	};
</script>

<form use:form class="space-y-6 p-1 flex flex-col h-full" data-testid="add-ip-form">
	<Slideover
		title="IPアドレスを追加"
		subtitle="アクセスを許可するIPアドレスまたはCIDRを追加"
		primaryButtonText="追加"
		primaryButtonTypeSubmit={true}
		onClose={handleClose}
		loading={$isSubmitting}
		data-testid="add-ip-slideover"
	>
		<div class="flex-grow space-y-6">
			<Input
				id="ip_address"
				name="ip_address"
				type="text"
				label="IPアドレス / CIDR"
				bind:value={$data.ip_address}
				error={$errors.ip_address?.[0]}
				required
				placeholder="例: 192.168.1.1 または 192.168.1.0/24"
				variant="underlined"
				data-testid="ip-address-input"
			/>
			<Input
				id="label"
				name="label"
				type="text"
				label="ラベル（任意）"
				bind:value={$data.label}
				error={$errors.label?.[0]}
				placeholder="例: オフィスネットワーク"
				variant="underlined"
				data-testid="label-input"
			/>
		</div>
	</Slideover>
</form>

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

		// Split IP and CIDR if present
		const parts = trimmed.split("/");
		const ipPart = parts[0];
		const cidrPart = parts[1];

		// Basic IPv4 validation
		const isIPv4 = /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/.test(ipPart);
		
		// Simple IPv6 validation - just check for basic format with colons
		const isIPv6 = /^[0-9a-fA-F:]+$/.test(ipPart) && ipPart.includes(":") && ipPart.length >= 2;

		if (!isIPv4 && !isIPv6) {
			return "IPアドレスの形式が正しくありません";
		}

		// If CIDR is present, validate it
		if (cidrPart !== undefined) {
			const cidr = parseInt(cidrPart, 10);
			if (isNaN(cidr)) {
				return "CIDR形式が正しくありません";
			}
			
			// Validate CIDR range
			if (isIPv4 && (cidr < 0 || cidr > 32)) {
				return "IPv4のCIDR範囲は0-32です";
			}
			if (isIPv6 && (cidr < 0 || cidr > 128)) {
				return "IPv6のCIDR範囲は0-128です";
			}
		}

		return null;
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

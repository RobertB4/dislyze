// Feature doc: docs/features/tenant-onboarding.md
import type { PageLoad } from "./$types";

export function load({ url }: Parameters<PageLoad>[0]) {
	const token = url.searchParams.get("token") ?? "";

	let email = "";
	let companyName = "";
	let userName = "";
	let ssoEnabled = false;

	if (token) {
		try {
			// JWT uses base64url encoding (RFC 7515), but atob() requires standard base64.
			// Convert base64url → base64 before decoding.
			const base64 = token.split(".")[1].replace(/-/g, "+").replace(/_/g, "/");
			const binaryString = atob(base64);
			const bytes = Uint8Array.from(binaryString, (c) => c.charCodeAt(0));
			const jsonPayload = new TextDecoder().decode(bytes);
			const payload = JSON.parse(jsonPayload);
			email = payload.email || "";
			companyName = payload.company_name || "";
			userName = payload.user_name || "";
			ssoEnabled = payload.sso?.enabled || false;
		} catch (e) {
			console.error("Failed to decode token:", e);
		}
	}

	return {
		token,
		email,
		companyName,
		userName,
		ssoEnabled
	};
}

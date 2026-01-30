import type { PageLoad } from "./$types";

export const load: PageLoad = ({ url }) => {
	const token = url.searchParams.get("token") ?? "";

	let email = "";
	let companyName = "";
	let userName = "";
	let ssoEnabled = false;

	if (token) {
		try {
			const base64 = token.split(".")[1];
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
};

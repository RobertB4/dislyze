// Feature doc: docs/features/tenant-onboarding.md
import type { PageLoad } from "./$types";

export function load({ url }: Parameters<PageLoad>[0]) {
	const token = url.searchParams.get("token");
	const inviterName = url.searchParams.get("inviter_name");
	const invitedEmail = url.searchParams.get("invited_email");

	return {
		token,
		inviterName,
		invitedEmail
	};
}

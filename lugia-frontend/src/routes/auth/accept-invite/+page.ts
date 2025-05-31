import type { PageLoad } from "./$types";

export const load: PageLoad = ({ url }) => {
	const token = url.searchParams.get("token");
	const inviterName = url.searchParams.get("inviter_name");
	const invitedEmail = url.searchParams.get("invited_email");

	return {
		token,
		inviterName,
		invitedEmail
	};
};

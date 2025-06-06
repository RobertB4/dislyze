import type { PageLoad } from "./$types";

export const load: PageLoad = () => {
	// Profile page doesn't need additional data loading since 
	// user/tenant data is available via me object from layout
	return {};
};
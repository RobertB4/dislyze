import { writable } from "svelte/store";

export type Me = {
	user_id: string;
	email: string;
	user_name: string;
	tenant_name: string;
	permissions: `${"tenant" | "users" | "roles"}.${"view" | "edit"}`[]; // array of {resource}.{action}, e.g. users.view
	enterprise_features: EnterpriseFeatures;
};

export type EnterpriseFeatures = {
	rbac: { enabled: boolean };
};

/**
 * Used to cache the response of /api/me.
 * Do not use this store directly in components.
 * Use the `me` property from PageData instead.
 */
export const meCache = writable<Me>(null as any);

/**
 * 	Used to force the layout to refresh meCache with fresh data
	Needed to ensure the updated name is reflected in the UI
 */
export const forceUpdateMeCache = writable(false);

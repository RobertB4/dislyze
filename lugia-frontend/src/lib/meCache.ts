import { writable } from "svelte/store";

export type Me = {
	user_id: string;
	email: string;
	user_name: string;
	tenant_name: string;
	permissions: `${"tenant" | "users" | "roles"}.${"view" | "edit"}`[]; // array of {resource}.{action}, e.g. users.view
	enterprise_features: EnterpriseFeatures;
};

type EnterpriseFeatures = {
	rbac: { enabled: boolean };
};

export function hasPermission(
	me: Me,
	permission: `${"tenant" | "users" | "roles"}.${"view" | "edit"}`
): boolean {
	if (me.permissions.includes(permission)) {
		return true;
	}

	if (permission.endsWith(".view")) {
		const editPermission = permission.replace(".view", ".edit") as typeof permission;
		return me.permissions.includes(editPermission);
	}

	return false;
}

export function hasFeature(me: Me, feature: keyof EnterpriseFeatures): boolean {
	return me.enterprise_features[feature].enabled;
}

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

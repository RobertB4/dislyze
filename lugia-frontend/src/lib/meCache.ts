import { writable } from "svelte/store";

export type Me = {
	user_id: string;
	email: string;
	user_name: string;
	permissions: string[]; // array of {resource}.{action}, e.g. users.update
	tenant_name: string;
};

/**
 * Check if the user has a specific permission
 * @param me - The Me object containing user permissions
 * @param permission - Permission in format "resource.action" (e.g., "users.view")
 * @returns boolean indicating if user has the permission
 */
export function hasPermission(me: Me, permission: string): boolean {
	return me.permissions.includes(permission);
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

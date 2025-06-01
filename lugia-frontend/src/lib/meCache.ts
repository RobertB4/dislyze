import { writable } from "svelte/store";

export type Me = {
	user_id: string;
	email: string;
	user_name: string | null;
	user_role: "admin" | "editor";
	tenant_name: string;
	tenant_plan: "none" | "basic" | "pro" | "enterprise";
};

/**
 * Used to cache the response of /api/me.
 * Do not use this store directly in components.
 * Use the `me` property from PageData instead.
 */
export const meCache = writable<Me>(null as any);

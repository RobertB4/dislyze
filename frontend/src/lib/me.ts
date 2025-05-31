import { writable } from "svelte/store";

export type Me = {
	user_id: string;
	email: string;
	user_name: string | null;
	user_role: "admin" | "editor";
	tenant_name: string;
	tenant_plan: "none" | "basic" | "pro" | "enterprise";
};

export const me = writable<Me>(null as any); // eslint-disable-line

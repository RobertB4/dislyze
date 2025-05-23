import { writable, derived } from "svelte/store";

export interface User {
	user_id: string;
	email: string;
	user_name: string | null;
	user_role: "admin" | "editor";
	tenant_name: string;
	tenant_plan: "none" | "basic" | "pro" | "enterprise";
}

export interface MeState {
	currentUser: User;
	isLoading: boolean;
}

export const meStore = writable<MeState>({
	currentUser: null as any,
	isLoading: true
});

export function setCurrentUser(user: User) {
	meStore.update((state) => ({ ...state, currentUser: user, isLoading: false }));
}

export function setMeStoreLoading(isLoading: boolean) {
	meStore.update((state) => ({ ...state, isLoading }));
}

export function resetMeStore() {
	meStore.set({ currentUser: null as any, isLoading: true });
}

export const me = derived(meStore, ($meStore) => $meStore.currentUser);

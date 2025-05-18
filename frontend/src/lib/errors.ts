import { writable } from "svelte/store";

interface ErrorState {
	statusCode: number | null;
	message: string | null;
}

function createErrorStore() {
	const { subscribe, set } = writable<ErrorState>({ statusCode: null, message: null });

	return {
		subscribe,
		setError: (statusCode: number, message: string) => set({ statusCode, message }),
		clearError: () => set({ statusCode: null, message: null }),
		reset: () => set({ statusCode: null, message: null }) // Alias for clearError if preferred
	};
}

export const errorStore = createErrorStore();

export class KnownError {
	constructor(private _message: string) {}

	get message(): string {
		return this._message;
	}
}

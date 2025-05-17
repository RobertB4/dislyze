import { writable } from 'svelte/store';
import { KnownError } from '$lib/errors';

export type ToastMode = 'success' | 'error' | 'info';

interface Toast {
	id: number;
	text: string;
	mode: ToastMode;
}

function createToastStore() {
	const { subscribe, update } = writable<Toast[]>([]);
	let nextId = 0;

	return {
		subscribe,
		show: (textOrError: string | KnownError, mode: ToastMode = 'info') => {
			const id = nextId++;
			const text = textOrError instanceof KnownError ? textOrError.message : textOrError;
			update((toasts) => [...toasts, { id, text, mode }]);
			return id;
		},
		showError: (error: unknown) => {
			const id = nextId++;
			const text = error instanceof KnownError ? error.message : '予期せぬエラーが発生しました';
			update((toasts) => [...toasts, { id, text, mode: 'error' }]);
			return id;
		},
		remove: (id: number) => {
			update((toasts) => toasts.filter((toast) => toast.id !== id));
		}
	};
}

export const toast = createToastStore();

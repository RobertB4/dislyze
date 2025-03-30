import { writable } from 'svelte/store';

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
		show: (text: string, mode: ToastMode = 'info') => {
			const id = nextId++;
			update((toasts) => [...toasts, { id, text, mode }]);
			return id;
		},
		remove: (id: number) => {
			update((toasts) => toasts.filter((toast) => toast.id !== id));
		}
	};
}

export const toast = createToastStore();

import type { PageLoad } from './$types';
import type { PageData } from './types';

export const load: PageLoad<PageData> = async () => {
	try {
		const response = await fetch('http://localhost:1337');
		const data = await response.text();
		return {
			message: data
		};
	} catch (error) {
		console.error('Error fetching from backend:', error);
		return {
			message: 'Error connecting to backend'
		};
	}
};

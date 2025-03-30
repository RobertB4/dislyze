import type { PageLoad } from './$types';
import { PUBLIC_API_URL } from '$env/static/public';

interface User {
	id: string;
	email: string;
	first_name: string;
	last_name: string;
}

export const load: PageLoad = async ({ fetch }) => {
	try {
		const response = await fetch(`${PUBLIC_API_URL}/users`, {
			credentials: 'include'
		});

		if (!response.ok) {
			throw new Error('Failed to fetch users');
		}

		const users: User[] = await response.json();
		return { users };
	} catch (error) {
		return {
			error: error instanceof Error ? error.message : 'An error occurred',
			users: [] as User[]
		};
	}
};

import type { PageLoad } from './$types';

interface User {
	id: string;
	email: string;
	first_name: string;
	last_name: string;
}

export const load: PageLoad = async ({ fetch }) => {
	try {
		const response = await fetch('http://localhost:1337/users', {
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

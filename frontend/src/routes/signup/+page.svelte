<script lang="ts">
	import { goto } from '$app/navigation';
	import { toast } from '$components/toast';
	import { createForm } from 'felte';
	import { PUBLIC_API_URL } from '$env/static/public';

	const { form, errors, data, isValid, isSubmitting } = createForm({
		initialValues: {
			company_name: '',
			user_name: '',
			email: '',
			password: '',
			password_confirm: ''
		},
		validate: (values) => {
			const errors: Record<string, string> = {};

			// Trim whitespace from all fields
			values.company_name = values.company_name.trim();
			values.user_name = values.user_name.trim();
			values.email = values.email.trim();
			values.password = values.password.trim();
			values.password_confirm = values.password_confirm.trim();

			// Check for empty or whitespace-only fields
			if (!values.company_name) {
				errors.company_name = 'Company name is required';
			}
			if (!values.user_name) {
				errors.user_name = 'User name is required';
			}
			if (!values.email) {
				errors.email = 'Email is required';
			} else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(values.email)) {
				errors.email = 'Invalid email format';
			}
			if (!values.password) {
				errors.password = 'Password is required';
			} else if (values.password.length < 8) {
				errors.password = 'Password must be at least 8 characters long';
			}
			if (!values.password_confirm) {
				errors.password_confirm = 'Please confirm your password';
			} else if (values.password !== values.password_confirm) {
				errors.password_confirm = 'Passwords do not match';
			}

			return errors;
		},
		onSubmit: async (values) => {
			try {
				const response = await fetch(`${PUBLIC_API_URL}/auth/signup`, {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json'
					},
					body: JSON.stringify(values),
					credentials: 'include'
				});

				const data = await response.json();
				console.log({ data });

				if (data.error) {
					throw new Error(data.error);
				}

				// Show success toast and redirect to dashboard
				toast.show('Account created successfully!', 'success');
				goto('/dashboard');
			} catch (err) {
				console.log({ err });
				const errorMessage = err instanceof Error ? err.message : 'An error occurred';

				toast.show(errorMessage, 'error');
			}
		}
	});
</script>

<main class="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
	<div class="max-w-md w-full space-y-8">
		<div>
			<h2 class="mt-6 text-center text-3xl font-extrabold text-gray-900">Create your account</h2>
			<p class="mt-2 text-center text-sm text-gray-600">
				Or
				<a href="/auth/login" class="font-medium text-indigo-600 hover:text-indigo-500">
					sign in to your account
				</a>
			</p>
		</div>

		<form class="mt-8 space-y-6" use:form>
			<div class="rounded-md shadow-sm space-y-4">
				<div>
					<label for="company_name" class="sr-only">Company Name</label>
					<input
						id="company_name"
						name="company_name"
						type="text"
						required
						bind:value={$data.company_name}
						class="appearance-none rounded-md relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
						placeholder="Company Name"
					/>
					{#if $errors.company_name}
						<p class="mt-1 text-sm text-red-600">{$errors.company_name}</p>
					{/if}
				</div>

				<div>
					<label for="user_name" class="sr-only">Full Name</label>
					<input
						id="user_name"
						name="user_name"
						type="text"
						required
						bind:value={$data.user_name}
						class="appearance-none rounded-md relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
						placeholder="Full Name"
					/>
					{#if $errors.user_name}
						<p class="mt-1 text-sm text-red-600">{$errors.user_name}</p>
					{/if}
				</div>

				<div>
					<label for="email" class="sr-only">Email address</label>
					<input
						id="email"
						name="email"
						type="email"
						required
						bind:value={$data.email}
						class="appearance-none rounded-md relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
						placeholder="Email address"
					/>
					{#if $errors.email}
						<p class="mt-1 text-sm text-red-600">{$errors.email}</p>
					{/if}
				</div>

				<div>
					<label for="password" class="sr-only">Password</label>
					<input
						id="password"
						name="password"
						type="password"
						required
						bind:value={$data.password}
						class="appearance-none rounded-md relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
						placeholder="Password"
					/>
					{#if $errors.password}
						<p class="mt-1 text-sm text-red-600">{$errors.password}</p>
					{/if}
				</div>

				<div>
					<label for="password_confirm" class="sr-only">Confirm Password</label>
					<input
						id="password_confirm"
						name="password_confirm"
						type="password"
						required
						bind:value={$data.password_confirm}
						class="appearance-none rounded-md relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
						placeholder="Confirm Password"
					/>
					{#if $errors.password_confirm}
						<p class="mt-1 text-sm text-red-600">{$errors.password_confirm}</p>
					{/if}
				</div>
			</div>

			<div>
				<button
					type="submit"
					disabled={!$isValid}
					class="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
				>
					{#if $isSubmitting}
						<span class="absolute left-0 inset-y-0 flex items-center pl-3">
							<svg
								class="animate-spin h-5 w-5 text-white"
								xmlns="http://www.w3.org/2000/svg"
								fill="none"
								viewBox="0 0 24 24"
							>
								<circle
									class="opacity-25"
									cx="12"
									cy="12"
									r="10"
									stroke="currentColor"
									stroke-width="4"
								></circle>
								<path
									class="opacity-75"
									fill="currentColor"
									d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
								></path>
							</svg>
						</span>
					{/if}
					Sign up
				</button>
			</div>
		</form>
	</div>
</main>

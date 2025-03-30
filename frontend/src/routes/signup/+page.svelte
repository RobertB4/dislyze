<script lang="ts">
	import { goto } from '$app/navigation';

	let formData = {
		company_name: '',
		user_name: '',
		email: '',
		password: '',
		password_confirm: ''
	};

	let error: string | null = null;
	let loading = false;

	async function handleSubmit() {
		loading = true;
		error = null;

		try {
			const response = await fetch('http://localhost:1337/auth/signup', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(formData),
				credentials: 'include'
			});

			const data = await response.json();

			if (!response.ok) {
				throw new Error(data.error || 'Failed to sign up');
			}

			// Redirect to dashboard on success
			goto('/dashboard');
		} catch (err) {
			error = err instanceof Error ? err.message : 'An error occurred';
		} finally {
			loading = false;
		}
	}
</script>

<main>
	<div class="signup-container">
		<div class="header">
			<h2>Create your account</h2>
			<p>
				Or
				<a href="/auth/login"> sign in to your account </a>
			</p>
		</div>

		<form on:submit|preventDefault={handleSubmit}>
			{#if error}
				<div class="error-message">
					<p>{error}</p>
				</div>
			{/if}

			<div class="form-fields">
				<div>
					<label for="company_name" class="sr-only">Company Name</label>
					<input
						id="company_name"
						name="company_name"
						type="text"
						required
						bind:value={formData.company_name}
						placeholder="Company Name"
					/>
				</div>

				<div>
					<label for="user_name" class="sr-only">Full Name</label>
					<input
						id="user_name"
						name="user_name"
						type="text"
						required
						bind:value={formData.user_name}
						placeholder="Full Name"
					/>
				</div>

				<div>
					<label for="email" class="sr-only">Email address</label>
					<input
						id="email"
						name="email"
						type="email"
						required
						bind:value={formData.email}
						placeholder="Email address"
					/>
				</div>

				<div>
					<label for="password" class="sr-only">Password</label>
					<input
						id="password"
						name="password"
						type="password"
						required
						bind:value={formData.password}
						placeholder="Password"
					/>
				</div>

				<div>
					<label for="password_confirm" class="sr-only">Confirm Password</label>
					<input
						id="password_confirm"
						name="password_confirm"
						type="password"
						required
						bind:value={formData.password_confirm}
						placeholder="Confirm Password"
					/>
				</div>
			</div>

			<div class="submit-container">
				<button type="submit" disabled={loading} class="submit-button">
					{#if loading}
						<span class="loading-spinner">
							<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
								<circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
								<path
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

<style>
	main {
		min-height: 100vh;
		display: flex;
		align-items: center;
		justify-content: center;
		background-color: #f9fafb;
		padding: 3rem 1rem;
	}

	.signup-container {
		max-width: 28rem;
		width: 100%;
	}

	.header {
		text-align: center;
		margin-bottom: 2rem;
	}

	h2 {
		margin-top: 1.5rem;
		font-size: 1.875rem;
		font-weight: 800;
		color: #111827;
	}

	.header p {
		margin-top: 0.5rem;
		font-size: 0.875rem;
		color: #4b5563;
	}

	.header a {
		font-weight: 500;
		color: #4f46e5;
		text-decoration: none;
	}

	.header a:hover {
		color: #4338ca;
	}

	form {
		margin-top: 2rem;
	}

	.error-message {
		background-color: #fef2f2;
		border: 1px solid #fecaca;
		border-radius: 0.375rem;
		padding: 1rem;
		margin-bottom: 1rem;
	}

	.error-message p {
		font-size: 0.875rem;
		color: #dc2626;
	}

	.form-fields {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		margin: -1px;
		overflow: hidden;
		clip: rect(0, 0, 0, 0);
		white-space: nowrap;
		border: 0;
	}

	input {
		width: 100%;
		padding: 0.5rem 0.75rem;
		border: 1px solid #d1d5db;
		border-radius: 0.375rem;
		font-size: 0.875rem;
		color: #111827;
	}

	input:focus {
		outline: none;
		border-color: #4f46e5;
		box-shadow: 0 0 0 2px rgba(79, 70, 229, 0.2);
	}

	.submit-container {
		margin-top: 1.5rem;
	}

	.submit-button {
		position: relative;
		width: 100%;
		display: flex;
		justify-content: center;
		padding: 0.5rem 1rem;
		border: none;
		border-radius: 0.375rem;
		font-size: 0.875rem;
		font-weight: 500;
		color: white;
		background-color: #4f46e5;
		cursor: pointer;
	}

	.submit-button:hover {
		background-color: #4338ca;
	}

	.submit-button:focus {
		outline: none;
		box-shadow:
			0 0 0 2px white,
			0 0 0 4px #4f46e5;
	}

	.submit-button:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.loading-spinner {
		position: absolute;
		left: 1rem;
		top: 50%;
		transform: translateY(-50%);
	}

	.loading-spinner svg {
		height: 1.25rem;
		width: 1.25rem;
		color: white;
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from {
			transform: rotate(0deg);
		}
		to {
			transform: rotate(360deg);
		}
	}
</style>

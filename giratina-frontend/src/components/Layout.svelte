<script
	lang="ts"
	generics="PromisesMap extends Record<string, Promise<any>> = Record<string, Promise<any>>"
>
	import { page } from "$app/state";
	import { EmptyAvatar, toast, safeGoto } from "@dislyze/zoroark";
	import { slide, fade } from "svelte/transition";
	import type { Snippet } from "svelte";
	import type { Me } from "@dislyze/zoroark";
	import { resolve } from "$app/paths";

	let isMobileNavigationOpen = $state(false);

	type ResolvedObject<T extends Record<string, Promise<any>>> = {
		[K in keyof T]: T[K] extends Promise<infer U> ? U : never;
	};

	type LayoutProps = {
		me: Me;
		pageTitle: string;
		promises?: PromisesMap;
		buttons?: Snippet;
		children: Snippet<[ResolvedObject<PromisesMap>]>;
		skeleton?: Snippet;
	};

	let { me, pageTitle, promises, buttons, children, skeleton }: LayoutProps = $props();

	function toggleMobileNavigation() {
		isMobileNavigationOpen = !isMobileNavigationOpen;
	}

	async function handleLogout() {
		try {
			const res = await fetch(`/api/auth/logout`, {
				method: "POST",
				credentials: "include"
			});

			if (!res.ok) {
				toast.showError("ログアウト中にエラーが発生しました。");
			}

			window.location.pathname = "/auth/login";
		} catch (logoutError) {
			console.error("Logout request failed:", logoutError);
			toast.showError("ログアウト中にエラーが発生しました。");
		}
	}

	const getResolvedPromises = $derived(async () => {
		try {
			if (!promises || Object.keys(promises).length === 0) {
				return {} as ResolvedObject<PromisesMap>;
			}
			const keys = Object.keys(promises);
			const promiseValues = Object.values(promises);

			const results = await Promise.all(promiseValues);

			const resolvedObject: Record<string, any> = {};
			keys.forEach((key, index) => {
				resolvedObject[key] = results[index];
			});
			return resolvedObject as ResolvedObject<PromisesMap>;
		} catch (e) {
			console.error("Error resolving promises in Layout:", e);
			let status = 500;
			let message = "処理中に予期せぬエラーが発生しました。";

			const err = e as {
				status?: number;
				message?: string;
				body?: { message?: string };
				location?: string;
			};

			if (err.status) {
				status = err.status;
			}

			if (err.message) {
				message = err.message;
			}

			if (err.body?.message) {
				message = err.body.message;
			}

			if (e instanceof Error) {
				message = e.message;
			}

			if (err.location) {
				Promise.resolve().then(() => {
					safeGoto(err.location!);
				});
				return {} as ResolvedObject<PromisesMap>;
			}

			// Use a microtask to ensure navigation happens after the current processing cycle
			Promise.resolve().then(() => {
				safeGoto(`/error?status=${status}&message=${encodeURIComponent(message)}`);
			});

			// Return an empty object. The navigation should ideally prevent children from rendering
			// or attempting to use potentially incomplete data.
			return {} as ResolvedObject<PromisesMap>;
		}
	});
</script>

{#if isMobileNavigationOpen}
	<div class="relative z-40 md:hidden" role="dialog" aria-modal="true">
		<div class="fixed inset-0 bg-gray-600 opacity-75" transition:fade={{ duration: 300 }}></div>

		<div class="fixed inset-0 z-40 flex" transition:slide={{ duration: 300, axis: "x" }}>
			<div class="relative flex w-full max-w-xs flex-1 flex-col bg-gray-800">
				<div class="absolute top-0 right-0 -mr-12 pt-2">
					<button
						type="button"
						onclick={toggleMobileNavigation}
						class="ml-1 flex h-10 w-10 items-center justify-center cursor-pointer rounded-full focus:outline-none focus:ring-2 focus:ring-inset focus:ring-white"
					>
						<span class="sr-only">Close sidebar</span>
						<!-- Heroicon name: outline/x-mark -->
						<svg
							class="h-6 w-6 text-white"
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
							stroke-width="1.5"
							stroke="currentColor"
							aria-hidden="true"
						>
							<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"></path>
						</svg>
					</button>
				</div>

				<div class="h-0 flex-1 overflow-y-auto pt-5 pb-4">
					<div class="flex flex-shrink-0 items-center px-4">
						<img class="h-8 w-auto" src="/logo.png" alt="dislyze logo" />
					</div>
					<nav class="mt-5 space-y-1 px-2">
						<a
							data-testid="navigation-home-mobile"
							href={resolve("/")}
							onclick={toggleMobileNavigation}
							class="text-gray-300 hover:bg-gray-700 hover:text-white group flex items-center px-2 py-2 mb-4 text-base rounded-md"
							class:bg-gray-900={page.route.id === "/"}
						>
							<!-- Heroicon name: building-office-2 -->
							<svg
								class="text-gray-400 group-hover:text-gray-300 mr-3 flex-shrink-0 h-6 w-6"
								class:text-gray-100={page.route.id?.includes("/data/companies")}
								xmlns="http://www.w3.org/2000/svg"
								fill="none"
								viewBox="0 0 24 24"
								stroke-width="1.5"
								stroke="currentColor"
								aria-hidden="true"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M2.25 21h19.5m-18-18v18m10.5-18v18m6-13.5V21M6.75 6.75h.75m-.75 3h.75m-.75 3h.75m3-6h.75m-.75 3h.75m-.75 3h.75M6.75 21v-3.375c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125V21M3 3h12m-.75 4.5H21m-3.75 3.75h.008v.008h-.008v-.008zm0 3h.008v.008h-.008v-.008zm0 3h.008v.008h-.008v-.008z"
								></path>
							</svg>
							テナント一覧
						</a>
					</nav>
				</div>

				<nav class="px-2 pb-2">
					<span
						role="menu"
						tabindex={6}
						data-testid="navigation-signout-mobile"
						onclick={toggleMobileNavigation}
						class="w-full text-gray-300 hover:bg-gray-700 hover:text-white group flex items-center px-2 py-2 text-base rounded-md cursor-pointer"
						onkeypress={() => {}}
					>
						<!-- Heroicon name: arrow-left-on-rectangle -->
						<svg
							class="text-gray-400 group-hover:text-gray-300 mr-4 flex-shrink-0 h-6 w-6"
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
							stroke-width="1.5"
							stroke="currentColor"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M15.75 9V5.25A2.25 2.25 0 0013.5 3h-6a2.25 2.25 0 00-2.25 2.25v13.5A2.25 2.25 0 007.5 21h6a2.25 2.25 0 002.25-2.25V15M12 9l-3 3m0 0l3 3m-3-3h12.75"
							></path>
						</svg>

						<span>ログアウト</span>
					</span>
				</nav>

				<div class="flex flex-shrink-0 bg-gray-700 p-4">
					<div class="flex items-center">
						<div>
							<EmptyAvatar />
						</div>
						<div class="ml-3">
							<p class="text-base font-medium text-white">{me.user_name}</p>
							<p class="text-sm font-medium text-gray-400 group-hover:text-gray-300 truncate">
								{me.email}
							</p>
						</div>
					</div>
				</div>
			</div>

			<div class="w-14 flex-shrink-0">
				<!-- Force sidebar to shrink to fit close icon -->
			</div>
		</div>
	</div>
{/if}
<!-- Static sidebar for desktop -->
<div class="hidden md:fixed md:inset-y-0 md:w-56 md:flex md:flex-col" style="transition: 1s ease;">
	<!-- Sidebar component, swap this element with another sidebar if you like -->
	<div class="flex min-h-0 flex-1 flex-col bg-gray-800">
		<div class="flex flex-1 flex-col overflow-y-auto pt-5 pb-4 overflow-hidden">
			<div class="flex flex-shrink-0 justify-between items-center h-8 px-4">
				<img class="h-8 w-auto" src="/logo.png" alt="dislyze logo" />
			</div>
			<nav class="mt-5 flex-1 space-y-1 px-2">
				<a
					data-testid="navigation-home"
					href={resolve("/")}
					class=" text-gray-300 hover:bg-gray-700 hover:text-white group flex items-center px-2 py-2 mb-4 text-sm font-medium rounded-md"
					class:bg-gray-900={page.route.id === "/"}
				>
					<!-- Heroicon name: building-office-2 -->
					<svg
						class="text-gray-400 group-hover:text-gray-300 mr-3 flex-shrink-0 h-6 w-6"
						class:text-gray-100={page.route.id?.includes("/data/companies")}
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="1.5"
						stroke="currentColor"
						aria-hidden="true"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M2.25 21h19.5m-18-18v18m10.5-18v18m6-13.5V21M6.75 6.75h.75m-.75 3h.75m-.75 3h.75m3-6h.75m-.75 3h.75m-.75 3h.75M6.75 21v-3.375c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125V21M3 3h12m-.75 4.5H21m-3.75 3.75h.008v.008h-.008v-.008zm0 3h.008v.008h-.008v-.008zm0 3h.008v.008h-.008v-.008z"
						></path>
					</svg>
					<span class="ml-2">テナント一覧</span>
				</a>
			</nav>
		</div>

		<nav class="px-2 pb-2">
			<button
				role="menu"
				tabindex={6}
				data-testid="navigation-signout"
				class=" text-gray-300 hover:bg-gray-700 w-full cursor-pointer hover:text-white group flex items-center px-2 py-2 text-sm font-medium rounded-md"
				onclick={handleLogout}
			>
				<!-- Heroicon name: arrow-left-on-rectangle -->
				<svg
					class="text-gray-400 group-hover:text-gray-300 mr-3 flex-shrink-0 h-6 w-6"
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="1.5"
					stroke="currentColor"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M15.75 9V5.25A2.25 2.25 0 0013.5 3h-6a2.25 2.25 0 00-2.25 2.25v13.5A2.25 2.25 0 007.5 21h6a2.25 2.25 0 002.25-2.25V15M12 9l-3 3m0 0l3 3m-3-3h12.75"
					></path>
				</svg>

				<span class="ml-2">ログアウト</span>
			</button>
		</nav>

		<div class="flex flex-shrink-0 bg-gray-700 px-2 py-4">
			<div class="flex items-center">
				<EmptyAvatar />

				<div class="ml-4 overflow-hidden">
					<p class="text-sm font-medium text-white h-4 mb-1">{me.user_name}</p>
					<p
						data-testid="layout-user-email"
						class="text-xs font-medium text-gray-300 group-hover:text-gray-200 h-4 truncate"
					>
						{me.email}
					</p>
				</div>
			</div>
		</div>
	</div>
</div>
<div class="flex flex-1 flex-col md:pl-56" style="transition: 1s ease;">
	<!-- <div class="sticky top-0 z-10 bg-gray-100 pl-1 pt-1 sm:pl-3 sm:pt-3 md:hidden">
      
    </div> -->
	<main class="flex-1">
		<div
			class="sticky top-0 flex justify-between bg-white shadow shadow-gray-300 z-10 py-3 mx-auto h-[62px]"
		>
			<div class="flex items-center px-4 sm:px-6 md:px-8">
				<button
					type="button"
					onclick={toggleMobileNavigation}
					class="-ml-0.5 -mt-0.5 mr-4 inline-flex cursor-pointer h-8 w-8 items-center justify-center rounded-md text-gray-500 hover:text-gray-900 focus:outline-none focus:ring-2 focus:ring-inset focus:ring-indigo-500 md:hidden"
				>
					<span class="sr-only">Open sidebar</span>
					<!-- Heroicon name: outline/bars-3 -->
					<svg
						class="h-6 w-6"
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="1.5"
						stroke="currentColor"
						aria-hidden="true"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5"
						></path>
					</svg>
				</button>

				<h1
					class="text-2xl font-semibold text-gray-900 pr-2 hidden md:block"
					data-testid="page-title"
				>
					{pageTitle}
				</h1>
			</div>
			<div class="px-4 sm:px-6 md:px-8">
				{#if buttons}
					{@render buttons()}
				{/if}
			</div>
		</div>
		<div class="py-6 px-4 sm:px-6 md:px-8">
			{#await getResolvedPromises()}
				{@render skeleton?.()}
			{:then resolvedData}
				{@render children(resolvedData)}
			{/await}
		</div>
	</main>
</div>

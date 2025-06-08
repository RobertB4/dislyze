<script
	lang="ts"
	generics="PromisesMap extends Record<string, Promise<any>> = Record<string, Promise<any>>"
>
	import { page } from "$app/state";
	import EmptyAvatar from "$components/EmptyAvatar.svelte";
	import { errorStore } from "$lib/errors";
	import { slide, fade } from "svelte/transition";
	import type { Snippet } from "svelte";
	import { safeGoto } from "$lib/routing";
	import type { Me } from "$lib/meCache";
	import { hasPermission } from "$lib/meCache";

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
				errorStore.setError(500, "処理中に予期せぬエラーが発生しました。");
			}

			window.location.pathname = "/auth/login";
		} catch (logoutError) {
			console.error("Logout request failed:", logoutError);
			errorStore.setError(500, "処理中に予期せぬエラーが発生しました。");
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
							href="/"
							onclick={toggleMobileNavigation}
							class="text-gray-300 hover:bg-gray-700 hover:text-white group flex items-center px-2 py-2 mb-4 text-base rounded-md"
							class:bg-gray-900={page.route.id === "/"}
						>
							<!--
                    Heroicon name: outline/home
    
                    Current: "text-gray-300", Default: "text-gray-400 group-hover:text-gray-300"
                  -->
							<svg
								class="text-gray-400 group-hover:text-gray-300 mr-4 flex-shrink-0 h-6 w-6"
								class:text-gray-100={page.route.id === "/"}
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
									d="M2.25 12l8.954-8.955c.44-.439 1.152-.439 1.591 0L21.75 12M4.5 9.75v10.125c0 .621.504 1.125 1.125 1.125H9.75v-4.875c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125V21h4.125c.621 0 1.125-.504 1.125-1.125V9.75M8.25 21h8.25"
								></path>
							</svg>
							ダッシュボード
						</a>

						<div
							class="border-0 bg-gray-500 mb-4"
							style="height:1px;margin-bottom:0.75rem!important;"
						></div>

						<a
							data-testid="navigation-custom-fields-mobile"
							href="/data/custom-fields"
							onclick={toggleMobileNavigation}
							class="text-gray-300 hover:bg-gray-700 hover:text-white group flex items-center px-2 py-2 text-base rounded-md"
							class:bg-gray-900={page.route.id?.includes("/data/custom-fields")}
						>
							<!-- Heroicon name: outline/table-cells -->
							<svg
								class="text-gray-400 group-hover:text-gray-300 mr-4 flex-shrink-0 h-6 w-6"
								class:text-gray-100={page.route.id?.includes("/data/custom-fields")}
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
									d="M3.375 19.5h17.25m-17.25 0a1.125 1.125 0 01-1.125-1.125M3.375 19.5h7.5c.621 0 1.125-.504 1.125-1.125m-9.75 0V5.625m0 12.75v-1.5c0-.621.504-1.125 1.125-1.125m18.375 2.625V5.625m0 12.75c0 .621-.504 1.125-1.125 1.125m1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125m0 3.75h-7.5A1.125 1.125 0 0112 18.375m9.75-12.75c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125m19.5 0v1.5c0 .621-.504 1.125-1.125 1.125M2.25 5.625v1.5c0 .621.504 1.125 1.125 1.125m0 0h17.25m-17.25 0h7.5c.621 0 1.125.504 1.125 1.125M3.375 8.25c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125m17.25-3.75h-7.5c-.621 0-1.125.504-1.125 1.125m8.625-1.125c.621 0 1.125.504 1.125 1.125v1.5c0 .621-.504 1.125-1.125 1.125m-17.25 0h7.5m-7.5 0c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125M12 10.875v-1.5m0 1.5c0 .621-.504 1.125-1.125 1.125M12 10.875c0 .621.504 1.125 1.125 1.125m-2.25 0c.621 0 1.125.504 1.125 1.125M13.125 12h7.5m-7.5 0c-.621 0-1.125.504-1.125 1.125M20.625 12c.621 0 1.125.504 1.125 1.125v1.5c0 .621-.504 1.125-1.125 1.125m-17.25 0h7.5M12 14.625v-1.5m0 1.5c0 .621-.504 1.125-1.125 1.125M12 14.625c0 .621.504 1.125 1.125 1.125m-2.25 0c.621 0 1.125.504 1.125 1.125m0 1.5v-1.5m0 0c0-.621.504-1.125 1.125-1.125m0 0h7.5"
								></path>
							</svg>
							メニュー1
						</a>

						<a
							data-testid="navigation-companies-mobile"
							href="/data/companies"
							onclick={toggleMobileNavigation}
							class="text-gray-300 hover:bg-gray-700 hover:text-white group flex items-center px-2 py-2 text-base rounded-md"
							class:bg-gray-900={page.route.id?.includes("/data/companies")}
						>
							<!-- Heroicon name: building-office-2 -->
							<svg
								class="text-gray-400 group-hover:text-gray-300 mr-4 flex-shrink-0 h-6 w-6"
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
							メニュー2
						</a>

						<a
							data-testid="navigation-users-mobile"
							href="/data/users"
							onclick={toggleMobileNavigation}
							class="text-gray-300 hover:bg-gray-700 hover:text-white group flex items-center px-2 py-2 text-base rounded-md"
							class:bg-gray-900={page.route.id?.includes("/data/users")}
						>
							<!-- Heroicon name: users -->
							<svg
								class="text-gray-400 group-hover:text-gray-300 mr-4 flex-shrink-0 h-6 w-6"
								class:text-gray-100={page.route.id?.includes("/data/users")}
								xmlns="http://www.w3.org/2000/svg"
								fill="none"
								viewBox="0 0 24 24"
								stroke-width="1.5"
								stroke="currentColor"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M15 19.128a9.38 9.38 0 002.625.372 9.337 9.337 0 004.121-.952 4.125 4.125 0 00-7.533-2.493M15 19.128v-.003c0-1.113-.285-2.16-.786-3.07M15 19.128v.106A12.318 12.318 0 018.624 21c-2.331 0-4.512-.645-6.374-1.766l-.001-.109a6.375 6.375 0 0111.964-3.07M12 6.375a3.375 3.375 0 11-6.75 0 3.375 3.375 0 016.75 0zm8.25 2.25a2.625 2.625 0 11-5.25 0 2.625 2.625 0 015.25 0z"
								></path>
							</svg>
							メニュー3
						</a>

						<a
							data-testid="navigation-segments-mobile"
							href="/data/segments"
							onclick={toggleMobileNavigation}
							class="text-gray-300 hover:bg-gray-700 hover:text-white group flex items-center px-2 py-2 text-base rounded-md"
							class:bg-gray-900={page.route.id?.includes("/data/segments")}
						>
							<!-- Heroicon name: chart-pie -->
							<svg
								class="text-gray-400 group-hover:text-gray-300 mr-4 flex-shrink-0 h-6 w-6"
								class:text-gray-100={page.route.id?.includes("/data/segments")}
								xmlns="http://www.w3.org/2000/svg"
								fill="none"
								viewBox="0 0 24 24"
								stroke-width="1.5"
								stroke="currentColor"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M10.5 6a7.5 7.5 0 107.5 7.5h-7.5V6z"
								></path>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M13.5 10.5H21A7.5 7.5 0 0013.5 3v7.5z"
								></path>
							</svg>
							メニュー4
						</a>
					</nav>
				</div>

				<nav class="px-2 pb-2">
					<a
						data-testid="navigation-settings-mobile"
						href="/settings/users"
						onclick={toggleMobileNavigation}
						class="text-gray-300 hover:bg-gray-700 hover:text-white group flex items-center px-2 py-2 text-base rounded-md"
						class:bg-gray-900={page.route.id?.includes("/settings")}
					>
						<!-- Heroicon name: outline/cog-6-tooth -->
						<svg
							class="text-gray-400 group-hover:text-gray-300 mr-4 flex-shrink-0 h-6 w-6"
							class:text-gray-100={page.route.id?.includes("/settings")}
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
							stroke-width="1.5"
							stroke="currentColor"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.24-.438.613-.431.992a6.759 6.759 0 010 .255c-.007.378.138.75.43.99l1.005.828c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.992a6.932 6.932 0 010-.255c.007-.378-.138-.75-.43-.99l-1.004-.828a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.214-1.281z"
							></path>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
							></path>
						</svg>
						設定
					</a>

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
					<a
						data-testid="navigation-avatar-mobile"
						href="/settings/profile"
						class="group block flex-shrink-0"
					>
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
					</a>
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
					href="/"
					class=" text-gray-300 hover:bg-gray-700 hover:text-white group flex items-center px-2 py-2 mb-4 text-sm font-medium rounded-md"
					class:bg-gray-900={page.route.id === "/"}
				>
					<!--
                  Heroicon name: outline/home
    
                  Current: "text-gray-300", Default: "text-gray-400 group-hover:text-gray-300"
                -->
					<svg
						class="text-gray-400 group-hover:text-gray-300 mr-3 flex-shrink-0 h-6 w-6"
						class:text-gray-100={page.route.id === "/"}
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
							d="M2.25 12l8.954-8.955c.44-.439 1.152-.439 1.591 0L21.75 12M4.5 9.75v10.125c0 .621.504 1.125 1.125 1.125H9.75v-4.875c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125V21h4.125c.621 0 1.125-.504 1.125-1.125V9.75M8.25 21h8.25"
						></path>
					</svg>
					<span class="ml-2">ダッシュボード</span>
				</a>

				<div
					class="border-0 bg-gray-500 mb-4"
					style="height:1px;margin-bottom:0.75rem!important;"
				></div>

				<a
					data-testid="navigation-custom-fields"
					href="/data/custom-fields"
					class=" text-gray-300 hover:bg-gray-700 hover:text-white group flex items-center px-2 py-2 text-sm font-medium rounded-md"
					class:bg-gray-900={page.route.id?.includes("/data/custom-fields")}
				>
					<!-- Heroicon name: outline/table-cells -->
					<svg
						class="text-gray-400 group-hover:text-gray-300 mr-3 flex-shrink-0 h-6 w-6"
						class:text-gray-100={page.route.id?.includes("/data/custom-fields")}
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
							d="M3.375 19.5h17.25m-17.25 0a1.125 1.125 0 01-1.125-1.125M3.375 19.5h7.5c.621 0 1.125-.504 1.125-1.125m-9.75 0V5.625m0 12.75v-1.5c0-.621.504-1.125 1.125-1.125m18.375 2.625V5.625m0 12.75c0 .621-.504 1.125-1.125 1.125m1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125m0 3.75h-7.5A1.125 1.125 0 0112 18.375m9.75-12.75c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125m19.5 0v1.5c0 .621-.504 1.125-1.125 1.125M2.25 5.625v1.5c0 .621.504 1.125 1.125 1.125m0 0h17.25m-17.25 0h7.5c.621 0 1.125.504 1.125 1.125M3.375 8.25c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125m17.25-3.75h-7.5c-.621 0-1.125.504-1.125 1.125m8.625-1.125c.621 0 1.125.504 1.125 1.125v1.5c0 .621-.504 1.125-1.125 1.125m-17.25 0h7.5m-7.5 0c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125M12 10.875v-1.5m0 1.5c0 .621-.504 1.125-1.125 1.125M12 10.875c0 .621.504 1.125 1.125 1.125m-2.25 0c.621 0 1.125.504 1.125 1.125M13.125 12h7.5m-7.5 0c-.621 0-1.125.504-1.125 1.125M20.625 12c.621 0 1.125.504 1.125 1.125v1.5c0 .621-.504 1.125-1.125 1.125m-17.25 0h7.5M12 14.625v-1.5m0 1.5c0 .621-.504 1.125-1.125 1.125M12 14.625c0 .621.504 1.125 1.125 1.125m-2.25 0c.621 0 1.125.504 1.125 1.125m0 1.5v-1.5m0 0c0-.621.504-1.125 1.125-1.125m0 0h7.5"
						></path>
					</svg>
					<span class="ml-2">メニュー1</span>
				</a>

				<a
					data-testid="navigation-companies"
					href="/data/companies"
					class=" text-gray-300 hover:bg-gray-700 hover:text-white group flex items-center px-2 py-2 text-sm font-medium rounded-md"
					class:bg-gray-900={page.route.id?.includes("/data/companies")}
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
					<span class="ml-2">メニュー2</span>
				</a>

				<a
					data-testid="navigation-users"
					href="/data/users"
					class=" text-gray-300 hover:bg-gray-700 hover:text-white group flex items-center px-2 py-2 text-sm font-medium rounded-md"
					class:bg-gray-900={page.route.id?.includes("/data/users")}
				>
					<!-- Heroicon name: users -->
					<svg
						class="text-gray-400 group-hover:text-gray-300 mr-3 flex-shrink-0 h-6 w-6"
						class:text-gray-100={page.route.id?.includes("/data/users")}
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="1.5"
						stroke="currentColor"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M15 19.128a9.38 9.38 0 002.625.372 9.337 9.337 0 004.121-.952 4.125 4.125 0 00-7.533-2.493M15 19.128v-.003c0-1.113-.285-2.16-.786-3.07M15 19.128v.106A12.318 12.318 0 018.624 21c-2.331 0-4.512-.645-6.374-1.766l-.001-.109a6.375 6.375 0 0111.964-3.07M12 6.375a3.375 3.375 0 11-6.75 0 3.375 3.375 0 016.75 0zm8.25 2.25a2.625 2.625 0 11-5.25 0 2.625 2.625 0 015.25 0z"
						></path>
					</svg>
					<span class="ml-2">メニュー3</span>
				</a>

				<a
					data-testid="navigation-segments"
					href="/data/segments"
					class=" text-gray-300 hover:bg-gray-700 hover:text-white group flex items-center px-2 py-2 text-sm font-medium rounded-md"
					class:bg-gray-900={page.route.id?.includes("/data/segments")}
				>
					<!-- Heroicon name: chart-pie -->
					<svg
						class="text-gray-400 group-hover:text-gray-300 mr-3 flex-shrink-0 h-6 w-6"
						class:text-gray-100={page.route.id?.includes("/data/segments")}
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="1.5"
						stroke="currentColor"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M10.5 6a7.5 7.5 0 107.5 7.5h-7.5V6z"
						></path>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M13.5 10.5H21A7.5 7.5 0 0013.5 3v7.5z"
						></path>
					</svg>

					<span class="ml-2">メニュー4</span>
				</a>
			</nav>
		</div>

		<nav class="px-2 pb-2">
			{#if hasPermission(me, "users.view")}
				<a
					data-testid="navigation-settings"
					href="/settings/users"
					class=" text-gray-300 hover:bg-gray-700 hover:text-white group flex items-center px-2 py-2 text-sm font-medium rounded-md"
					class:bg-gray-900={page.route.id?.includes("/settings")}
				>
					<!-- Heroicon name: outline/cog-6-tooth -->
					<svg
						class="text-gray-400 group-hover:text-gray-300 mr-3 flex-shrink-0 h-6 w-6"
						class:text-gray-100={page.route.id?.includes("/settings")}
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
							d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.24-.438.613-.431.992a6.759 6.759 0 010 .255c-.007.378.138.75.43.99l1.005.828c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.992a6.932 6.932 0 010-.255c.007-.378-.138-.75-.43-.99l-1.004-.828a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.214-1.281z"
						></path>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
						></path>
					</svg>
					<span class="ml-2">設定</span>
				</a>
			{/if}

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
			<a
				data-testid="navigation-avatar"
				href="/settings/profile"
				class="group block w-full flex-shrink-0"
			>
				<div class="flex items-center">
					<EmptyAvatar />

					<div class="ml-4 overflow-hidden">
						<p class="text-sm font-medium text-white h-4 mb-1">{me.user_name}</p>
						<p class="text-xs font-medium text-gray-300 group-hover:text-gray-200 h-4 truncate">
							{me.email}
						</p>
					</div>
				</div></a
			>
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

<script lang="ts">
	import Button from "$components/Button.svelte";
	import { onDestroy, onMount, type Snippet } from "svelte";
	import { slide } from "svelte/transition";

	let {
		onClose,
		title,
		subtitle = "",
		primaryButtonText = "",
		primaryButtonTypeSubmit = false,
		onPrimaryClick = undefined,
		loading = false,
		widthClass = "",
		"data-testid": dataTestid,
		children
	}: {
		onClose: () => void;
		title: string;
		subtitle?: string;
		primaryButtonText?: string;
		primaryButtonTypeSubmit?: boolean;
		onPrimaryClick?: () => void;
		loading?: boolean;
		widthClass?: string;
		"data-testid"?: string;
		children: Snippet;
	} = $props();

	let widthCss = $derived(widthClass ? widthClass : "max-w-2xl");

	onMount(() => {
		document.body.style.overflow = "hidden";
	});

	onDestroy(() => {
		document.body.style.overflow = "visible";
	});

	const handleClose = () => {
		if (loading) {
			return;
		}
		onClose();
	};
</script>

<div
	class="relative z-10"
	aria-labelledby="Slideover"
	role="dialog"
	aria-modal="true"
	data-testid={dataTestid}
>
	<div class="fixed inset-0 bg-gray-500 opacity-75"></div>

	<div class="fixed inset-0 overflow-hidden">
		<div class="absolute inset-0 overflow-hidden">
			<div class="pointer-events-none fixed inset-y-0 right-0 flex pl-10 sm:pl-16 max-w-full">
				<div
					class="pointer-events-auto w-screen {widthCss}"
					transition:slide|global={{ axis: "x", duration: 750 }}
					data-testid={dataTestid ? `${dataTestid}-panel` : ""}
				>
					<div class="flex h-full flex-col overflow-y-scroll bg-white shadow-xl">
						<div class="flex-1">
							<!-- Header -->
							<div class="bg-gray-50 px-4 py-6 sm:px-6">
								<div class="flex items-start justify-between space-x-3">
									<div class="space-y-1">
										<h2 class="text-lg font-medium text-gray-900" id="slide-over-title">{title}</h2>
										<p class="text-sm text-gray-500">{subtitle}</p>
									</div>
									<div class="flex h-7 items-center">
										<button
											type="button"
											onclick={handleClose}
											class="text-gray-400 hover:text-gray-500 cursor-pointer"
											data-testid={dataTestid ? `${dataTestid}-close-button` : ""}
										>
											<span class="sr-only">Close panel</span>
											<!-- Heroicon name: outline/x-mark -->
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
													d="M6 18L18 6M6 6l12 12"
												></path>
											</svg>
										</button>
									</div>
								</div>
							</div>

							<div
								class="px-4 sm:px-6 sm:py-5"
								data-testid={dataTestid ? `${dataTestid}-content` : ""}
							>
								{@render children()}
							</div>
						</div>

						<!-- Action buttons -->
						<div class="flex-shrink-0 border-t border-gray-200 px-4 py-5 sm:px-6">
							<div class="flex justify-start space-x-3">
								{#if primaryButtonText}
									<Button
										type={primaryButtonTypeSubmit ? "submit" : "button"}
										{loading}
										onclick={onPrimaryClick}
										data-testid={dataTestid ? `${dataTestid}-primary-button` : ""}
										>{primaryButtonText}</Button
									>
								{/if}
								<Button
									variant="secondary"
									onclick={handleClose}
									data-testid={dataTestid ? `${dataTestid}-cancel-button` : ""}>キャンセル</Button
								>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>

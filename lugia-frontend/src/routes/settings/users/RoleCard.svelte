<script lang="ts">
	import Tooltip from "$components/Tooltip.svelte";
	import type { RoleInfo } from "./+page";

	let {
		role,
		isSelected = false,
		onclick,
		class: customClass = "",
		"data-testid": dataTestid
	}: {
		role: RoleInfo;
		isSelected?: boolean;
		onclick?: () => void;
		class?: string;
		"data-testid"?: string;
	} = $props();

	const handleClick = () => {
		onclick?.();
	};

	const handleKeydown = (event: KeyboardEvent) => {
		if (event.key === "Enter" || event.key === " ") {
			event.preventDefault();
			onclick?.();
		}
	};
</script>

<div
	class="relative border rounded-lg p-4 cursor-pointer transition-all duration-200 {isSelected
		? 'border-blue-500 bg-blue-50 shadow-md'
		: 'border-gray-200 bg-white hover:border-gray-300 hover:shadow-sm'} {customClass}"
	onclick={handleClick}
	onkeydown={handleKeydown}
	role="button"
	tabindex="0"
	data-testid={dataTestid}
	aria-pressed={isSelected}
>
	{#if isSelected}
		<div class="absolute top-3 right-3">
			<svg
				class="h-5 w-5 text-blue-500"
				fill="currentColor"
				viewBox="0 0 20 20"
				xmlns="http://www.w3.org/2000/svg"
			>
				<path
					fill-rule="evenodd"
					d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
					clip-rule="evenodd"
				></path>
			</svg>
		</div>
	{/if}

	<div class="space-y-3">
		<div>
			<h3 class="text-sm font-semibold text-gray-900">{role.name}</h3>
			{#if role.description}
				<p class="text-sm text-gray-600 mt-1">{role.description}</p>
			{/if}
		</div>

		{#if role.permissions && role.permissions.length > 0}
			<div>
				<h4 class="text-xs font-medium text-gray-700 mb-2">権限:</h4>
				<div class="grid grid-cols-2 gap-x-4">
					<ul class="space-y-1">
						{#each role.permissions.slice(0, 3) as permission (permission.id)}
							<li class="text-xs text-gray-600 flex items-center">
								<svg
									class="h-3 w-3 text-gray-400 mr-2 flex-shrink-0"
									fill="currentColor"
									viewBox="0 0 20 20"
								>
									<path
										fill-rule="evenodd"
										d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
										clip-rule="evenodd"
									></path>
								</svg>
								{permission.description}
							</li>
						{/each}
					</ul>
					<ul class="space-y-1">
						{#each role.permissions.slice(3, 5) as permission (permission.id)}
							<li class="text-xs text-gray-600 flex items-center">
								<svg
									class="h-3 w-3 text-gray-400 mr-2 flex-shrink-0"
									fill="currentColor"
									viewBox="0 0 20 20"
								>
									<path
										fill-rule="evenodd"
										d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
										clip-rule="evenodd"
									></path>
								</svg>
								{permission.description}
							</li>
						{/each}
						{#if role.permissions.length > 5}
							<li class="text-xs text-gray-600 flex items-center">
								<svg
									class="h-3 w-3 text-gray-400 mr-2 flex-shrink-0"
									fill="currentColor"
									viewBox="0 0 20 20"
								>
									<path
										fill-rule="evenodd"
										d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
										clip-rule="evenodd"
									></path>
								</svg>
								<Tooltip position="right">
									{#snippet content()}
										<div class="space-y-1">
											{#each (role.permissions || []).slice(5) as permission (permission.id)}
												<div class="text-xs">{permission.description}</div>
											{/each}
										</div>
									{/snippet}

									<span
										class="cursor-help border-b border-dotted border-gray-300 flex items-center"
									>
										他{role.permissions.length - 5}件
										<svg class="h-5 w-5 text-gray-800 ml-1" fill="currentColor" viewBox="0 0 20 20">
											<path
												fill-rule="evenodd"
												d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-8-3a1 1 0 00-.867.5 1 1 0 11-1.731-1A3 3 0 0113 8a3.001 3.001 0 01-2 2.83V11a1 1 0 11-2 0v-1a1 1 0 011-1 1 1 0 100-2zm0 8a1 1 0 100-2 1 1 0 000 2z"
												clip-rule="evenodd"
											></path>
										</svg>
									</span>
								</Tooltip>
							</li>
						{/if}
					</ul>
				</div>
			</div>
		{:else}
			<div>
				<h4 class="text-xs font-medium text-gray-700 mb-2">権限:</h4>
				<p class="text-xs text-gray-500">権限なし</p>
			</div>
		{/if}
	</div>
</div>

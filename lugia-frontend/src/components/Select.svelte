<script lang="ts">
	import { slide } from "svelte/transition";
	import { flip } from "svelte/animate";

	type Option = { value: string; label: string };

	let {
		label,
		options,
		value = $bindable(),
		id,
		name,
		placeholder = "選択してください",
		class: customClass = ""
	}: {
		label: string;
		options: Option[];
		value?: string;
		id: string;
		name?: string;
		placeholder?: string;
		class?: string;
	} = $props();

	let isOpen = $state(false);
	let buttonElement: HTMLButtonElement | undefined = $state();
	let listElement: HTMLUListElement | undefined = $state();

	const selectedLabel = $derived(options.find((opt) => opt.value === value)?.label ?? placeholder);

	function selectOption(optionValue: string) {
		value = optionValue;
		isOpen = false;
		buttonElement?.focus();
	}

	function toggleDropdown() {
		isOpen = !isOpen;
	}

	function handleClickOutside(event: MouseEvent) {
		if (
			isOpen &&
			buttonElement &&
			!buttonElement.contains(event.target as Node) &&
			listElement &&
			!listElement.contains(event.target as Node)
		) {
			isOpen = false;
		}
	}

	function handleKeydown(event: KeyboardEvent) {
		if (!isOpen && event.key !== "Enter" && event.key !== " ") return;

		if (event.key === "Enter" || event.key === " ") {
			if (!isOpen) {
				event.preventDefault();
				isOpen = true;
				return;
			}
		}

		if (!listElement) return;
		const items: HTMLLIElement[] = Array.from(listElement.querySelectorAll('li[role="option"]'));
		const activeElement = document.activeElement as HTMLLIElement;
		let currentIndex = items.indexOf(activeElement);

		switch (event.key) {
			case "Escape":
				isOpen = false;
				buttonElement?.focus();
				break;
			case "ArrowDown":
				event.preventDefault();
				if (!isOpen) isOpen = true;
				currentIndex = (currentIndex + 1) % items.length;
				items[currentIndex]?.focus();
				break;
			case "ArrowUp":
				event.preventDefault();
				if (!isOpen) isOpen = true;
				currentIndex = (currentIndex - 1 + items.length) % items.length;
				items[currentIndex]?.focus();
				break;
			case "Enter":
			case " ":
				if (isOpen && activeElement && activeElement.dataset.value) {
					event.preventDefault();
					selectOption(activeElement.dataset.value);
				}
				break;
			case "Home":
				if (isOpen) {
					event.preventDefault();
					items[0]?.focus();
				}
				break;
			case "End":
				if (isOpen) {
					event.preventDefault();
					items[items.length - 1]?.focus();
				}
				break;
			default:
				if (isOpen && event.key.length === 1 && !event.ctrlKey && !event.metaKey) {
					event.preventDefault();
					const char = event.key.toLowerCase();
					const startIndex = currentIndex >= 0 ? (currentIndex + 1) % items.length : 0;
					for (let i = 0; i < items.length; i++) {
						const itemIndex = (startIndex + i) % items.length;
						const item = items[itemIndex];
						if (item.textContent?.toLowerCase().startsWith(char)) {
							item.focus();
							break;
						}
					}
				}
				break;
		}
	}
</script>

<svelte:window onclick={handleClickOutside} />

<div class={customClass}>
	<label for={id} class="block text-sm/6 font-medium text-gray-900">{label}</label>
	<div class="relative mt-2">
		<button
			bind:this={buttonElement}
			{id}
			type="button"
			{name}
			class="grid w-full cursor-pointer grid-cols-1 rounded-md bg-white py-1.5 pr-2 pl-3 text-left text-gray-900 outline-1 -outline-offset-1 outline-gray-300 focus:outline-2 focus:-outline-offset-2 focus:outline-orange-600 sm:text-sm/6"
			aria-haspopup="listbox"
			aria-expanded={isOpen}
			aria-labelledby="{id}-label"
			onclick={toggleDropdown}
			onkeydown={handleKeydown}
		>
			<span
				id="{id}-label"
				class="col-start-1 row-start-1 truncate pr-6 {value ? '' : 'text-gray-500'}"
				>{selectedLabel}</span
			>
			<svg
				class="col-start-1 row-start-1 size-5 self-center justify-self-end text-gray-500 sm:size-4"
				viewBox="0 0 16 16"
				fill="currentColor"
				aria-hidden="true"
			>
				<path
					fill-rule="evenodd"
					d="M5.22 10.22a.75.75 0 0 1 1.06 0L8 11.94l1.72-1.72a.75.75 0 1 1 1.06 1.06l-2.25 2.25a.75.75 0 0 1-1.06 0l-2.25-2.25a.75.75 0 0 1 0-1.06ZM10.78 5.78a.75.75 0 0 1-1.06 0L8 4.06 6.28 5.78a.75.75 0 0 1-1.06-1.06l2.25-2.25a.75.75 0 0 1 1.06 0l2.25 2.25a.75.75 0 0 1 0 1.06Z"
					clip-rule="evenodd"
				/>
			</svg>
		</button>

		{#if isOpen}
			<ul
				bind:this={listElement}
				transition:slide={{ duration: 150 }}
				class="absolute z-10 mt-1 max-h-60 w-full overflow-auto rounded-md bg-white py-1 text-base shadow-lg ring-1 ring-black/5 focus:outline-none sm:text-sm"
				tabindex="-1"
				role="listbox"
				aria-labelledby="{id}-label"
				aria-activedescendant={value && options.find((opt) => opt.value === value)
					? id + "-option-" + value
					: undefined}
				onkeydown={handleKeydown}
			>
				{#each options as option (option.value)}
					{@const isSelected = value === option.value}
					{@const isFocused = false}
					<li
						id="{id}-option-{option.value}"
						role="option"
						aria-selected={isSelected}
						tabindex="-1"
						data-value={option.value}
						class="relative cursor-pointer select-none py-2 pr-9 pl-3 {isSelected
							? 'font-semibold text-orange-600'
							: 'text-gray-900'} {isFocused ? 'bg-orange-50' : ''} hover:bg-orange-50"
						onclick={() => selectOption(option.value)}
						onkeydown={() => {}}
						onmouseenter={(e) => (e.target as HTMLLIElement).focus()}
						animate:flip={{ duration: 200 }}
					>
						<span class="block truncate {isSelected ? 'font-semibold' : 'font-normal'}"
							>{option.label}</span
						>
						{#if isSelected}
							<span class="absolute inset-y-0 right-0 flex items-center pr-4 text-orange-600">
								<svg class="size-5" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
									<path
										fill-rule="evenodd"
										d="M16.704 4.153a.75.75 0 0 1 .143 1.052l-8 10.5a.75.75 0 0 1-1.127.075l-4.5-4.5a.75.75 0 0 1 1.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 0 1 1.05-.143Z"
										clip-rule="evenodd"
									/>
								</svg>
							</span>
						{/if}
					</li>
				{/each}
			</ul>
		{/if}
	</div>
</div>

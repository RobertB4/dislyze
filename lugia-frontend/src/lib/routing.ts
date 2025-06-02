import { goto } from "$app/navigation";

/**
 * Blurs the active element before navigating to the given URL.
 * This is a workaround for a bug in Svelte 5 where the active element is not blurred in time when navigating.
 * @param url - The URL to navigate to.
 */
export async function safeGoto(url: string, options?: Parameters<typeof goto>[1]): Promise<void> {
	// svelte 5s rendering bugs when using goto while an input is focused
	const activeElement = document.activeElement as (Element & { blur?: () => void }) | null;
	if (activeElement) {
		activeElement.blur?.();
	}

	await goto(url, options);
}

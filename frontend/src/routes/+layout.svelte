<script lang="ts">
	import "../app.css";
	import ToastContainer from "$components/Toast/ToastContainer.svelte";
	import { page } from "$app/state";
	import { meStore, setCurrentUser, setMeStoreLoading } from "$lib/stores/meStore";
	import type { LayoutData } from "./$types";
	import { onMount } from "svelte";

	export let data: LayoutData; // Contains { initialUser: User | null }

	// This reactive block updates the meStore when `data.initialUser` from +layout.ts changes.
	// It runs after the `load` function completes and `data` is populated.
	$: {
		if (data && data.initialUser) {
			setCurrentUser(data.initialUser);
		}
	}

	// TODO: not working
	// This reactive block handles ensuring the spinner stops if an error page is displayed.
	$: {
		if (page.error && $meStore.isLoading) {
			setMeStoreLoading(false);
		}
	}

	// onMount: For the very first load, especially if the above reactive blocks might not immediately
	// set isLoading to false (e.g., if load resolves but data.initialUser is undefined for a moment).
	onMount(() => {
		// If the store is still loading after mount and the reactive blocks haven't set it,
		// and there's no page error, it implies data loading might have completed without initialUser
		// or an edge case. Ensure loading is false.
		if ($meStore.isLoading && !page.error) {
			if (data && data.hasOwnProperty("initialUser")) {
				// This case should have been caught by the reactive block above, but as a safeguard:
				if ($meStore.isLoading && data.initialUser) {
					// Double check, as setCurrentUser sets it to false
					setCurrentUser(data.initialUser);
				}
			} else {
				// No initialUser in data after mount, and no error page. Stop loading.
				console.log(
					"+layout.svelte: onMount - Fallback: no initialUser, no page error. Setting loading false."
				);
				setMeStoreLoading(false);
			}
		}
	});
</script>

<ToastContainer />
<slot />

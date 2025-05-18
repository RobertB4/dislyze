<script lang="ts">
	import "../app.css";
	import ToastContainer from "$components/Toast/ToastContainer.svelte";
	import GlobalErrorDisplay from "$components/GlobalErrorDisplay.svelte";
</script>

<ToastContainer />
<GlobalErrorDisplay />
<slot />

<script lang="ts">
	import { adminAuth } from "$lib/admin/AdminAuth.svelte";
	let { children } = $props();

	adminAuth.requireAuth();
</script>

<svelte:head></svelte:head>

{@render children()}

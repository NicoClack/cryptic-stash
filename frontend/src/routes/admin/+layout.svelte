<script lang="ts">
	import { AdminAuthState, setAdminAuth } from "$lib/admin/AdminAuth.svelte";
	let { children } = $props();

	const adminAuth = new AdminAuthState();
	adminAuth.requireAuth();
	setAdminAuth(adminAuth);
</script>

<svelte:head></svelte:head>

{@render children()}

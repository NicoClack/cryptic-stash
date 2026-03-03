<script lang="ts">
	import "../app.css";
	let { children } = $props();
</script>

<svelte:head />

<div class="dark min-h-screen bg-background text-foreground">
	{@render children()}
</div>

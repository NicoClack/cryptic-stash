<script lang="ts">
	import Label from "$lib/components/ui/label/label.svelte";

	let { children, ...restProps } = $props();
</script>

<Label {...restProps}>
	{@render children?.()}
</Label>

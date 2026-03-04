<script lang="ts">
	import { cn } from "$lib/utils";

	let { class: className, children, ...restProps } = $props();
</script>

<legend class={cn("text-sm font-medium", className)} {...restProps}>
	{@render children?.()}
</legend>

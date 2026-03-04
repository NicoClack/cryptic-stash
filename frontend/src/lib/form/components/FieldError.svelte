<script lang="ts">
	import { cn } from "$lib/utils";

	let { class: className, children, ...restProps } = $props();
</script>

<div class={cn("text-sm text-destructive", className)} {...restProps}>
	{@render children?.()}
</div>

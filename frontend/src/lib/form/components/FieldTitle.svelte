<script lang="ts">
	import { cn } from "$lib/utils";

	let { class: className, children, ...restProps } = $props();
</script>

<div class={cn("text-sm leading-none font-medium", className)} {...restProps}>
	{@render children?.()}
</div>

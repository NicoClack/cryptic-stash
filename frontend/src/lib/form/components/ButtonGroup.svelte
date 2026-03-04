<script lang="ts">
	import { cn } from "$lib/utils";

	let { class: className, children, ...restProps } = $props();
</script>

<div class={cn("flex items-center gap-2", className)} {...restProps}>
	{@render children?.()}
</div>

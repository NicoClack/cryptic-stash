<script lang="ts">
	import { cn } from "$lib/utils";

	let { class: className, children, ...restProps } = $props();
</script>

<div class={cn("grid gap-2", className)} {...restProps}>
	{@render children?.()}
</div>

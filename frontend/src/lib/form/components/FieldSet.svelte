<script lang="ts">
	import { cn } from "$lib/utils";

	let { class: className, children, ...restProps } = $props();
</script>

<fieldset class={cn("grid gap-4", className)} {...restProps}>
	{@render children?.()}
</fieldset>

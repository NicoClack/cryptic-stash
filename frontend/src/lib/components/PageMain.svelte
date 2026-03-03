<script lang="ts">
	import { cn, type WithElementRef } from "$lib/utils.js";
	import type { HTMLAttributes } from "svelte/elements";

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLElement>> = $props();
</script>

<main
	bind:this={ref}
	class={cn("mx-auto w-full max-w-3xl space-y-6 px-6 py-10", className)}
	{...restProps}
>
	{@render children?.()}
</main>

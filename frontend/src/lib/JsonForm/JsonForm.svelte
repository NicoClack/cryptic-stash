<script lang="ts">
	import type { JSONSchema4 } from "json-schema";

	const {
		schema,
		value,
		onSubmit,
	}: {
		schema: JSONSchema4;
		value: Record<string, any>;
		onSubmit: () => void | Promise<void>;
	} = $props();

	// TODO
</script>

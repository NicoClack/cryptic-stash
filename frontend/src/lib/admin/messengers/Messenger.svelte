<script lang="ts">
	const {
		name,
		enabled,
		options,
		optionsSchema,
	}: {
		name: string;
		enabled: boolean;
		options: Record<string, any>;
		optionsSchema: Record<string, any>;
	} = $props();
</script>

<div>
	<h3>{name} ({enabled ? "Enabled" : "Disabled"})</h3>
	{JSON.stringify(options)}
</div>

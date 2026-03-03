<script lang="ts">
	import { resolve } from "$app/paths";
	import type { AdminEnvVars } from "$lib/admin/setup";
	import { Button } from "$lib/components/ui/button";
	import { Label } from "$lib/components/ui/label";
	import { RadioGroup, RadioGroupItem } from "$lib/components/ui/radio-group";
	import { Textarea } from "$lib/components/ui/textarea";

	const {
		adminEnvVars,
	}: {
		adminEnvVars: AdminEnvVars;
	} = $props();

	type DisplayMode = "env" | "json";
	let displayMode = $state<DisplayMode>("env");

	const formattedVars = $derived.by(() => {
		if (displayMode === "env") {
			return Object.entries(adminEnvVars.envVars)
				.map(([key, value]) => `${key}=${JSON.stringify(value)}`)
				.join("\n");
		} else {
			return JSON.stringify(adminEnvVars.envVars, null, 2);
		}
	});
</script>

<section class="space-y-6">
	<h2 class="text-2xl text-balance font-semibold tracking-tight">
		Step 4 of 4: Use the Generated Environment Variables
	</h2>
	<p class="text-sm text-muted-foreground md:text-base">
		One last step for the environment setup, update your server to use these environment variables:
	</p>
	<RadioGroup class="grid gap-2" bind:value={displayMode}>
		<div class="flex items-center gap-2">
			<RadioGroupItem id="displayMode-env" value="env" />
			<Label for="displayMode-env">View as .env</Label>
		</div>
		<div class="flex items-center gap-2">
			<RadioGroupItem id="displayMode-json" value="json" />
			<Label for="displayMode-json">View as JSON</Label>
		</div>
	</RadioGroup>
	<Textarea class="min-h-72 font-mono" readonly rows={15} value={formattedVars} />

	<p class="text-sm text-muted-foreground md:text-base">
		Once your server has restarted, click Next.
	</p>
	<Button href={resolve("/")}>Next</Button>
</section>

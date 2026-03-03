<script lang="ts">
	import { resolve } from "$app/paths";
	import type { AdminEnvVars } from "$lib/admin/setup";

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
	<h2>Step 4 of 4: Use the Generated Environment Variables</h2>
	<p>
		One last step for the environment setup, update your server to use these environment variables:
	</p>
	<div class="flex flex-wrap gap-4">
		<label class="inline-flex items-center gap-2">
			<input
				type="radio"
				name="displayMode"
				value="env"
				checked={displayMode === "env"}
				onchange={() => {
					displayMode = "env";
				}}
			/>
			View as .env
		</label>
		<label class="inline-flex items-center gap-2">
			<input
				type="radio"
				name="displayMode"
				value="json"
				checked={displayMode === "json"}
				onchange={() => {
					displayMode = "json";
				}}
			/>
			View as JSON
		</label>
	</div>
	<textarea class="min-h-72 font-mono" readonly rows="15" value={formattedVars}></textarea>

	<p>Once your server has restarted, click Next.</p>
	<a
		class="inline-flex items-center justify-center rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
		href={resolve("/")}>Next</a
	>
</section>

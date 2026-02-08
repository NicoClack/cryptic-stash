<script lang="ts">
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

<main>
	<h3>Step 4 of 4: Use the Generated Environment Variables</h3>
	<p>
		One last step for the environment setup, update your server to use these environment variables:
	</p>
	<label>
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
	<label>
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
	</label> <br />
	<textarea readonly rows="15" cols="80" value={formattedVars}></textarea> <br />

	<p>Once you're server has restarted, click Next.</p>
	<a href="/">Next</a>
</main>

<style>
</style>

<script lang="ts">
	import { PUBLIC_API_DOMAIN } from "$env/static/public";

	const {
		onComplete,
	}: {
		onComplete: (headerName: string) => unknown;
	} = $props();

	let echoHeadersUrlObj = new URL(
		PUBLIC_API_DOMAIN + "/api/v1/setup/echo-headers/",
		window.location.origin,
	);
	let isLoading = $state(false);
	let headerName = $state("");

	async function handleSubmit(event: Event) {
		event.preventDefault();
		if (isLoading) return;
		isLoading = true;

		await onComplete(headerName.replace(/\s/g, ""));
		isLoading = false;
	}
</script>

<main>
	<h3>Step 3 of 4: Proxy Config</h3>
	<p>
		Please use Postman, curl, Node.js or another non-browser HTTP client to make a GET request to
		<span class="echo-headers-url">
			{echoHeadersUrlObj.toString()}
		</span>. Look for headers that contain your public IP address. Once you find a candidate, try
		setting that header in your request to some other IP like 42.42.42.42 and make the request
		again. If this overwrote the proxy's header or they were combined, try another header. Once you
		have a header that can't be spoofed by the client, enter its name below.
	</p>
	<form onsubmit={handleSubmit}>
		<label>
			Header name:
			<input bind:value={headerName} type="text" name="header-name" />
		</label> <br />
		<p>Leave blank if there's no proxy.</p>
		<button type="submit" disabled={isLoading}>Next</button>
	</form>
</main>

<style>
	.echo-headers-url {
		font-family: monospace;
		background-color: #f0f0f0;
		padding: 0.2em 0.4em;
		border-radius: 4px;
	}
</style>

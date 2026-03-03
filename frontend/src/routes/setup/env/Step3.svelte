<script lang="ts">
	import { PUBLIC_API_DOMAIN } from "$env/static/public";
	import { Button } from "$lib/components/ui/button";
	import { Input } from "$lib/components/ui/input";
	import { Label } from "$lib/components/ui/label";

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

<section class="space-y-6">
	<h2 class="text-2xl text-balance font-semibold tracking-tight">Step 3 of 4: Proxy Config</h2>
	<p class="text-sm text-muted-foreground md:text-base">
		Please use Postman, curl, Node.js or another non-browser HTTP client to make a GET request to
		<span class="rounded-sm bg-muted px-1 py-0.5 font-mono text-foreground">
			{echoHeadersUrlObj.toString()}
		</span>. Look for headers that contain your public IP address. Once you find a candidate, try
		setting that header in your request to some other IP like 42.42.42.42 and make the request
		again. If this overwrote the proxy's header or they were combined, try another header. Once you
		have a header that can't be spoofed by the client, enter its name below.
	</p>
	<form class="space-y-4" onsubmit={handleSubmit}>
		<Label>
			Header name
			<Input bind:value={headerName} type="text" name="header-name" />
		</Label>
		<p class="text-sm text-muted-foreground md:text-base">Leave blank if there's no proxy.</p>
		<Button type="submit" disabled={isLoading}>Next</Button>
	</form>
</section>

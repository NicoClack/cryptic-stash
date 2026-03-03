<script lang="ts">
	import { fetchJson } from "$lib/api";
	import { Button } from "$lib/components/ui/button";
	import { Input } from "$lib/components/ui/input";
	import { Label } from "$lib/components/ui/label";
	import QRCode from "qrcode";

	const {
		totpURL,
		totpSecret,
		onComplete,
	}: {
		onComplete: () => unknown;
		totpURL: string;
		totpSecret: string;
	} = $props();

	let qrcodeUrlPromise = $derived(QRCode.toDataURL(totpURL));
	let isLoading = $state(false);
	let totpCode = $state("");

	async function handleSubmit(event: Event) {
		event.preventDefault();
		if (isLoading) return;
		isLoading = true;

		const response = await fetchJson(fetch, "/api/v1/setup/check-totp/", {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ code: totpCode.replace(/\s/g, ""), secret: totpSecret }),
		});
		if (response.redirecting || !response.ok) {
			isLoading = false;
			response.throwForStatus();
			return;
		}
		await onComplete();
		isLoading = false;
	}
</script>

<section class="space-y-6">
	<h2 class="text-2xl text-balance font-semibold tracking-tight">Step 2 of 4: Setup 2FA</h2>
	<p class="text-sm text-muted-foreground md:text-base">
		Please scan this QR code in your authenticator app (e.g., Google Authenticator, Authy) and enter
		the 2FA code you see.
	</p>
	<div class="h-25 w-25 overflow-hidden rounded-md border border-border bg-card p-2">
		{#await qrcodeUrlPromise then qrcodeUrl}
			<img class="h-full w-full" alt="TOTP QR Code" src={qrcodeUrl} width="100" height="100" />
		{:catch}
			<p class="text-sm text-muted-foreground md:text-base">Unable to generate QR code</p>
		{/await}
	</div>
	<a
		target="_blank"
		rel="external"
		href={totpURL}
		class="text-primary underline-offset-4 outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
	>
		I have a TOTP app on this device
	</a>
	<form class="space-y-4" onsubmit={handleSubmit}>
		<Label>
			2FA Code
			<Input
				bind:value={totpCode}
				required
				type="text"
				id="otp"
				name="otp"
				inputmode="numeric"
				pattern="[0-9\s]*"
				autocomplete="one-time-code"
			/>
		</Label>
		<Button type="submit" disabled={isLoading}>Next</Button>
	</form>
</section>

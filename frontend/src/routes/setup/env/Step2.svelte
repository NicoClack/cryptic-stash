<script lang="ts">
	import { fetchJson } from "$lib/api";
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

<main>
	<h3>Step 2 of 4: Setup 2FA</h3>
	<p>
		Please scan this QR code in your authenticator app (e.g., Google Authenticator, Authy) and enter
		the 2FA code you see.
	</p>
	<div class="qrcode-container">
		{#await qrcodeUrlPromise then qrcodeUrl}
			<img class="qrcode" alt="TOTP QR Code" src={qrcodeUrl} width="100" height="100" />
		{:catch}
			<p>Unable to generate QR code</p>
		{/await}
	</div>
	<a target="_blank" rel="external" href={totpURL}>I have a TOTP app on this device</a> <br />
	<form onsubmit={handleSubmit}>
		<label>
			2FA Code:
			<input
				bind:value={totpCode}
				required
				type="text"
				id="otp"
				name="otp"
				inputmode="numeric"
				pattern="[0-9\s]*"
				autocomplete="one-time-code"
			/>
		</label> <br />
		<button type="submit" disabled={isLoading}>Next</button>
	</form>
</main>

<style>
	.qrcode-container {
		width: 100px;
		height: 100px;
	}
	.qrcode {
		width: 100%;
		height: 100%;
	}
</style>

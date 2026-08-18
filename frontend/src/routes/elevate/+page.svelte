<script lang="ts">
	// TODO: un-AI this

	import { resolve } from "$app/paths";
	import { page } from "$app/state";
	import { fetchUserJson } from "$lib/api";
	import { userAuth } from "$lib/auth/UserAuth.svelte";
	import { Button } from "$lib/components/ui/button";
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "$lib/components/ui/card";
	import { decodeBase64UrlFormat, encodeBase64UrlFormat } from "$lib/utils";

	userAuth.requireAuth();
	if (userAuth.sudoMode) {
		redirectAfterElevation();
	}

	let isLoading = $state(false);
	let hasStarted = $state(false);
	let requestError = $state<string | null>(null);

	interface PublicKeyCredentialRequestOptionsJSON {
		challenge: string;
		timeout?: number;
		rpId?: string;
		allowCredentials?: {
			id: string;
			type: string;
			transports?: string[];
		}[];
		userVerification?: UserVerificationRequirement;
		extensions?: AuthenticationExtensionsClientInputs;
	}

	async function handleElevate() {
		if (isLoading) return;
		isLoading = true;
		requestError = null;
		hasStarted = true;

		try {
			const startResp = await fetchUserJson(fetch, "/api/v1/auth/sudo/start/", {
				method: "POST",
			});

			// If already elevated, redirect immediately
			if (startResp.status === 409) {
				redirectAfterElevation();
				return;
			}

			if (!startResp.ok) {
				requestError = "Failed to start elevation. Please try again.";
				isLoading = false;
				return;
			}

			const { publicKey, webAuthnSessionId } = startResp.data as {
				publicKey: PublicKeyCredentialRequestOptionsJSON;
				webAuthnSessionId: string;
			};

			const credentialOptions = {
				...publicKey,
				challenge: decodeBase64UrlFormat(publicKey.challenge),
				allowCredentials: publicKey.allowCredentials?.map((c) => ({
					...c,
					id: decodeBase64UrlFormat(c.id),
					type: c.type as PublicKeyCredentialType,
					transports: c.transports as AuthenticatorTransport[] | undefined,
				})),
			} satisfies PublicKeyCredentialRequestOptions;

			let credential: PublicKeyCredential;
			try {
				credential = (await navigator.credentials.get({
					publicKey: credentialOptions,
				})) as PublicKeyCredential;
			} catch {
				requestError = "Passkey authentication was cancelled or failed. Please try again.";
				isLoading = false;
				return;
			}

			if (!credential) {
				requestError = "No credential returned. Please try again.";
				isLoading = false;
				return;
			}

			const assertionResponse = credential.response as AuthenticatorAssertionResponse;
			const finishResponse = await fetchUserJson(fetch, "/api/v1/auth/sudo/finish/", {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
				},
				body: JSON.stringify({
					id: credential.id,
					type: credential.type,
					rawId: encodeBase64UrlFormat(credential.rawId),
					response: {
						clientDataJSON: encodeBase64UrlFormat(assertionResponse.clientDataJSON),
						authenticatorData: encodeBase64UrlFormat(assertionResponse.authenticatorData),
						signature: encodeBase64UrlFormat(assertionResponse.signature),
						userHandle: assertionResponse.userHandle
							? encodeBase64UrlFormat(assertionResponse.userHandle)
							: undefined,
					},
					webAuthnSessionId,
				}),
			});

			if (!finishResponse.ok) {
				const errorCode = finishResponse.data?.errors?.[0]?.code;
				if (errorCode === "NEITHER_PASSKEY_SUDO_ELIGIBLE") {
					// TODO: this error was removed
					requestError =
						"None of your passkeys are eligible for sudo mode elevation. Please use a passkey that has sudo mode access enabled.";
				} else {
					requestError = finishResponse.data?.errors?.[0]?.message ?? "Elevation failed. Please try again.";
				}
				isLoading = false;
				return;
			}

			// Success! Update sudo mode and redirect
			userAuth.elevate();
			redirectAfterElevation();
		} finally {
			isLoading = false;
		}
	}

	function redirectAfterElevation() {
		const redirectTo = page.url.searchParams.get("redirectTo") || "/";
		const urlObj = new URL(redirectTo, location.origin);
		if (urlObj.origin === location.origin) {
			window.location.href = urlObj.toString();
		} else {
			window.location.href = resolve("/");
		}
	}

	function handleCancel() {
		const redirectTo = page.url.searchParams.get("redirectTo") || "/";
		const urlObj = new URL(redirectTo, location.origin);
		if (urlObj.origin === location.origin) {
			window.location.href = urlObj.toString();
		} else {
			window.location.href = resolve("/");
		}
	}
</script>

<main class="mx-auto w-full max-w-md space-y-6 px-6 py-10">
	<h1 class="text-center text-3xl">
		<a
			href={resolve("/")}
			class="text-primary underline-offset-4 outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
			>Cryptic Stash</a
		>
	</h1>

	<Card>
		<CardHeader>
			<CardTitle>Sudo Mode Required</CardTitle>
			<CardDescription>
				You need to authenticate with an eligible passkey to enter sudo mode before proceeding.
			</CardDescription>
		</CardHeader>
		<CardContent class="space-y-4">
			{#if requestError}
				<p class="text-sm text-destructive">{requestError}</p>
			{/if}

			{#if !hasStarted}
				<p class="text-sm text-muted-foreground">
					Sudo mode allows you to manage your passkeys and perform sensitive account actions. Please
					authenticate with a passkey that has sudo mode access enabled.
				</p>
				<Button onclick={handleElevate} disabled={isLoading} class="w-full">
					{isLoading ? "Authenticating..." : "Authenticate to Elevate"}
				</Button>
			{:else if isLoading}
				<p class="text-sm text-muted-foreground">Follow the prompts on your device to authenticate...</p>
			{/if}

			<Button onclick={handleCancel} variant="ghost" class="w-full" disabled={isLoading}>Cancel</Button>
		</CardContent>
	</Card>
</main>

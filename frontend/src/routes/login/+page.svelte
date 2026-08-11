<script lang="ts">
	import { fetchJson } from "$lib/api";
	import { userAuth } from "$lib/auth/UserAuth.svelte";
	import PageMain from "$lib/components/PageMain.svelte";
	import { Button } from "$lib/components/ui/button";
	import {
		Card,
		CardContent,
		CardDescription,
		CardHeader,
		CardTitle,
	} from "$lib/components/ui/card";
	import { decodeBase64UrlFormat, encodeBase64UrlFormat } from "$lib/utils";

	let isLoading = $state(false);
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

	async function handleLogin() {
		if (isLoading) return;
		isLoading = true;
		requestError = null;

		try {
			const startResp = await fetchJson(fetch, "/api/v1/auth/login/start/", {
				method: "POST",
			});
			if (!startResp.ok) {
				requestError = "Failed to start login. Please try again.";
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
				return;
			}

			if (!credential) {
				requestError = "No credential returned. Please try again.";
				return;
			}

			const assertionResponse = credential.response as AuthenticatorAssertionResponse;
			const finishResponse = await fetchJson(fetch, "/api/v1/auth/login/finish/", {
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
				requestError =
					finishResponse.data?.errors?.[0]?.message ?? "Login failed. Please try again.";
				return;
			}

			const { userId, token, username, isSudo: isSudoMode } = finishResponse.data as {
				userId: string;
				token: string;
				username: string;
				isSudo: boolean;
			};
			userAuth.login(token, userId, username, isSudoMode);
		} finally {
			isLoading = false;
		}
	}
</script>

<PageMain>
	<Card>
		<CardHeader>
			<CardTitle>Login</CardTitle>
			<CardDescription>Sign in with your passkey</CardDescription>
		</CardHeader>
		<CardContent>
			{#if requestError}
				<p class="text-sm text-destructive mb-4">{requestError}</p>
			{/if}
			<Button onclick={handleLogin} disabled={isLoading} class="w-full">
				{#if isLoading}
					<p>Loading</p>
				{/if}
				Login with Passkey
			</Button>
		</CardContent>
	</Card>
</PageMain>

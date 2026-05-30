<script lang="ts">
	import { resolve } from "$app/paths";
	import { page } from "$app/state";
	import { fetchJson, type JsonResponse } from "$lib/api";
	import PageMain from "$lib/components/PageMain.svelte";
	import { Button } from "$lib/components/ui/button";
	import { Card, CardContent, CardHeader, CardTitle } from "$lib/components/ui/card";
	import { decodeBase64UrlFormat, encodeBase64UrlFormat } from "$lib/utils";

	interface InviteResponse {
		email: string;
		expiresAt: string;
	}

	interface WebAuthnOptionsResponse {
		publicKey: PublicKeyCredentialCreationOptionsJSON;
	}

	interface PublicKeyCredentialCreationOptionsJSON {
		rp: { id: string; name: string };
		user: { id: string; name: string; displayName: string };
		challenge: string;
		pubKeyCredParams: { type: string; alg: number }[];
		timeout?: number;
		excludeCredentials?: { id: string; type: string; transports?: string[] }[];
		authenticatorSelection?: Record<string, unknown>;
		attestation?: string;
	}

	const inviteCode = page.url.searchParams.get("code")?.trim() ?? "";
	const inviteId = page.params.id ?? "";

	let isLoadingLink = $state(true);
	let isCreating = $state(false);
	let requestError = $state<string | null>(null);
	let successMessage = $state<string | null>(null);

	let email = $state("");
	let credentialName = $state("");

	function getErrorMessage(response: JsonResponse): string {
		const firstError = response.data?.errors?.[0];
		if (firstError?.message) {
			return String(firstError.message);
		}
		return `Request failed with status ${response.status}`;
	}

	function getAuthHeaders(): HeadersInit {
		if (!inviteCode) return {};
		return { Authorization: `Bearer ${inviteCode}` };
	}

	async function loadInvite() {
		requestError = null;
		successMessage = null;
		isLoadingLink = true;

		if (!inviteCode) {
			requestError = "Missing invite code. Use the full invite link from your admin.";
			isLoadingLink = false;
			return;
		}

		try {
			const response = await fetchJson(fetch, `/api/v1/invites/${encodeURIComponent(inviteId)}`, {
				headers: getAuthHeaders(),
			});
			if (!response.ok) {
				requestError = getErrorMessage(response);
				return;
			}

			const data = response.data as InviteResponse;
			if (!email && data.email) {
				email = data.email;
			}
		} finally {
			isLoadingLink = false;
		}
	}

	async function handleCreateAccount() {
		if (isCreating) return;

		requestError = null;
		successMessage = null;
		credentialName = credentialName.trim();
		if (!credentialName) {
			requestError = "Enter a name for this passkey.";
			return;
		}

		if (!window.PublicKeyCredential) {
			requestError = "Your browser does not support passkeys. Please use a modern browser.";
			return;
		}

		isCreating = true;
		try {
			const optionsResponse = await fetchJson(
				fetch,
				`/api/v1/invites/${encodeURIComponent(inviteId)}/generate-options`,
				{
					method: "POST",
					headers: getAuthHeaders(),
				},
			);
			if (!optionsResponse.ok) {
				requestError = getErrorMessage(optionsResponse);
				return;
			}

			const { publicKey } = optionsResponse.data as WebAuthnOptionsResponse;

			const credentialOptions = {
				...publicKey,
				attestation: publicKey.attestation as AttestationConveyancePreference | undefined,
				pubKeyCredParams: publicKey.pubKeyCredParams.map((param) => ({
					...param,
					type: param.type as "public-key",
				})),
				challenge: decodeBase64UrlFormat(publicKey.challenge),
				user: {
					...publicKey.user,
					id: decodeBase64UrlFormat(publicKey.user.id),
				},
				excludeCredentials: publicKey.excludeCredentials?.map((c) => ({
					...c,
					id: decodeBase64UrlFormat(c.id),
					type: c.type as PublicKeyCredentialType,
					transports: c.transports as AuthenticatorTransport[] | undefined,
				})),
			} satisfies PublicKeyCredentialCreationOptions;

			let credential: PublicKeyCredential;
			try {
				credential = (await navigator.credentials.create({
					publicKey: credentialOptions,
				})) as PublicKeyCredential;
			} catch {
				requestError = "Passkey creation was cancelled or failed. Please try again.";
				return;
			}

			if (!credential) {
				requestError = "No credential returned. Please try again.";
				return;
			}

			const attestationResponse = credential.response as AuthenticatorAttestationResponse;
			const createResponse = await fetchJson(
				fetch,
				`/api/v1/invites/${encodeURIComponent(inviteId)}/create-user`,
				{
					method: "POST",
					headers: {
						"Content-Type": "application/json",
						...getAuthHeaders(),
					},
					body: JSON.stringify({
						id: credential.id,
						type: credential.type,
						rawId: encodeBase64UrlFormat(credential.rawId),
						response: {
							clientDataJSON: encodeBase64UrlFormat(attestationResponse.clientDataJSON),
							attestationObject: encodeBase64UrlFormat(attestationResponse.attestationObject),
							transports: attestationResponse.getTransports?.() ?? [],
						},
						credentialName,
					}),
				},
			);
			if (!createResponse.ok) {
				requestError = getErrorMessage(createResponse);
				return;
			}

			successMessage = "Account created successfully. Your passkey has been registered.";
		} finally {
			isCreating = false;
		}
	}

	loadInvite();
</script>

<PageMain class="max-w-3xl">
	<h1 class="text-center text-3xl">
		<a
			href={resolve("/")}
			class="text-primary underline-offset-4 outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
			>Cryptic Stash</a
		>
	</h1>

	<Card>
		<CardHeader>
			<CardTitle>Create your account</CardTitle>
		</CardHeader>
		<CardContent class="space-y-4">
			{#if requestError}
				<div
					class="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive"
				>
					{requestError}
				</div>
			{/if}

			{#if successMessage}
				<div
					class="rounded-md border border-emerald-500/40 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-300"
				>
					{successMessage}
				</div>
			{/if}

			{#if isLoadingLink}
				<p class="text-sm text-muted-foreground">Validating invite...</p>
			{:else if !successMessage}
				<div class="space-y-3">
					<p class="text-sm text-muted-foreground">
						Creating account for <span class="font-medium text-foreground">{email}</span>
					</p>
					<p class="text-sm text-muted-foreground">
						You'll register a passkey to securely access your stash. Your device will prompt you to
						authenticate.
					</p>
					<label class="block space-y-2 text-sm">
						<span class="text-muted-foreground">Passkey name</span>
						<input
							required
							bind:value={credentialName}
							type="text"
							maxlength="64"
							class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
						/>
					</label>
					<Button onclick={handleCreateAccount} disabled={isCreating} class="w-full">
						{isCreating ? "Registering passkey..." : "Create account with passkey"}
					</Button>
				</div>
			{/if}
		</CardContent>
	</Card>
</PageMain>

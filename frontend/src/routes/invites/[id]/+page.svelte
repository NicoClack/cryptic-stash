<script lang="ts">
	import { resolve } from "$app/paths";
	import { page } from "$app/state";
	import { fetchJson, type JsonResponse } from "$lib/api";
	import PageMain from "$lib/components/PageMain.svelte";
	import { Button } from "$lib/components/ui/button";
	import { Card, CardContent, CardHeader, CardTitle } from "$lib/components/ui/card";

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

	let isLoadingLink = $state(true);
	let isCreating = $state(false);
	let requestError = $state<string | null>(null);
	let successMessage = $state<string | null>(null);

	let email = $state("");

	function getErrorMessage(response: JsonResponse): string {
		const firstError = response.data?.errors?.[0];
		if (firstError?.message) {
			return String(firstError.message);
		}
		return `Request failed with status ${response.status}`;
	}

	function getInviteCode(): string {
		return page.url.searchParams.get("code")?.trim() ?? "";
	}

	function getInviteId(): string {
		return page.params.id ?? "";
	}

	function getAuthHeaders(): HeadersInit {
		const code = getInviteCode();
		if (!code) return {};
		return { Authorization: `Bearer ${code}` };
	}

	function base64urlDecode(str: string): Uint8Array {
		const padded = str + "=".repeat((4 - (str.length % 4)) % 4);
		const binary = atob(padded.replace(/-/g, "+").replace(/_/g, "/"));
		return Uint8Array.from(binary, (c) => c.charCodeAt(0));
	}

	function base64urlEncode(buffer: ArrayBuffer): string {
		const bytes = new Uint8Array(buffer);
		let binary = "";
		for (const byte of bytes) binary += String.fromCharCode(byte);
		return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=/g, "");
	}

	async function loadInvite() {
		requestError = null;
		successMessage = null;
		isLoadingLink = true;

		const inviteId = getInviteId();
		const code = getInviteCode();
		if (!code) {
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

		if (!window.PublicKeyCredential) {
			requestError = "Your browser does not support passkeys. Please use a modern browser.";
			return;
		}

		const inviteId = getInviteId();
		isCreating = true;
		try {
			const optionsResponse = await fetchJson(
				fetch,
				`/api/v1/invites/${encodeURIComponent(inviteId)}/webauthn-options`,
				{ headers: getAuthHeaders() },
			);
			if (!optionsResponse.ok) {
				requestError = getErrorMessage(optionsResponse);
				return;
			}

			const { publicKey } = optionsResponse.data as WebAuthnOptionsResponse;

			const credentialOptions: PublicKeyCredentialCreationOptions = {
				...publicKey,
				challenge: base64urlDecode(publicKey.challenge),
				user: {
					...publicKey.user,
					id: base64urlDecode(publicKey.user.id),
				},
				excludeCredentials: publicKey.excludeCredentials?.map((c) => ({
					...c,
					id: base64urlDecode(c.id),
					type: c.type as PublicKeyCredentialType,
				})),
			};

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
			const credentialJSON = {
				id: credential.id,
				type: credential.type,
				rawId: base64urlEncode(credential.rawId),
				response: {
					clientDataJSON: base64urlEncode(attestationResponse.clientDataJSON),
					attestationObject: base64urlEncode(attestationResponse.attestationObject),
					transports: attestationResponse.getTransports?.() ?? [],
				},
			};

			const createResponse = await fetchJson(
				fetch,
				`/api/v1/invites/${encodeURIComponent(inviteId)}/create-user`,
				{
					method: "POST",
					headers: {
						"Content-Type": "application/json",
						...getAuthHeaders(),
					},
					body: JSON.stringify({ credential: credentialJSON }),
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
					<Button onclick={handleCreateAccount} disabled={isCreating} class="w-full">
						{isCreating ? "Registering passkey..." : "Create account with passkey"}
					</Button>
				</div>
			{/if}
		</CardContent>
	</Card>
</PageMain>

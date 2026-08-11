<script lang="ts">
	// TODO: un-AI this

	import { fetchUserJson } from "$lib/api";
	import { userAuth } from "$lib/auth/UserAuth.svelte";
	import { Button } from "$lib/components/ui/button";
	import {
		Card,
		CardContent,
		CardDescription,
		CardHeader,
		CardTitle,
	} from "$lib/components/ui/card";
	import { Checkbox } from "$lib/components/ui/checkbox";
	import { Input } from "$lib/components/ui/input";
	import { Label } from "$lib/components/ui/label";
	import { decodeBase64UrlFormat, encodeBase64UrlFormat } from "$lib/utils";

	interface Props {
		onSuccess?: () => void;
		isSecondGroup?: boolean;
	}

	let { onSuccess, isSecondGroup = false }: Props = $props();

	userAuth.requireSudoMode(); // TODO: should this go in the page instead?

	let isLoading = $state(false);
	let requestError = $state<string | null>(null);
	let passKeyName = $state("");
	let allowSudo = $state(true);
	let hasStarted = $state(false);

	interface PublicKeyCredentialCreationOptionsJSON {
		rp: {
			id?: string;
			name: string;
		};
		user: {
			id: string;
			name: string;
			displayName: string;
		};
		challenge: string;
		pubKeyCredParams: { type: string; alg: number }[];
		timeout?: number;
		attestation?: AttestationConveyancePreference;
		authenticatorSelection?: {
			authenticatorAttachment?: AuthenticatorAttachment;
			residentKey?: ResidentKeyRequirement;
			userVerification?: UserVerificationRequirement;
		};
		extensions?: AuthenticationExtensionsClientInputs;
		signal?: AbortSignal;
		excludeCredentials?: {
			id: string;
			type: string;
			transports?: string[];
		}[];
	}

	async function handleStartRegistration() {
		if (isLoading || !passKeyName.trim()) return;
		isLoading = true;
		requestError = null;

		try {
			const startResp = await fetchUserJson(fetch, "/api/v1/self/passkeys/register/start/", {
				method: "POST",
			});
			if (!startResp.ok) {
				requestError = "Failed to start passkey registration. Please try again.";
				return;
			}

			const { publicKey, webAuthnSessionId } = startResp.data as {
				publicKey: PublicKeyCredentialCreationOptionsJSON;
				webAuthnSessionId: string;
			};

			const credentialOptions = {
				...publicKey,
				challenge: decodeBase64UrlFormat(publicKey.challenge),
				user: {
					...publicKey.user,
					id: decodeBase64UrlFormat(publicKey.user.id),
				},
				pubKeyCredParams: publicKey.pubKeyCredParams.map((p) => ({
					type: p.type as PublicKeyCredentialType,
					alg: p.alg,
				})),
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
			} catch (e: any) {
				requestError = "Passkey registration was cancelled or failed. Please try again.";
				isLoading = false;
				throw e;
			}

			if (!credential) {
				requestError = "No credential returned. Please try again.";
				isLoading = false;
				return;
			}

			const attestationResponse = credential.response as AuthenticatorAttestationResponse;
			const finishResponse = await fetchUserJson(fetch, "/api/v1/self/passkeys/register/finish/", {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
				},
				body: JSON.stringify({
					id: credential.id,
					type: credential.type,
					rawId: encodeBase64UrlFormat(credential.rawId),
					response: {
						clientDataJSON: encodeBase64UrlFormat(attestationResponse.clientDataJSON),
						attestationObject: encodeBase64UrlFormat(attestationResponse.attestationObject),
					},
					webAuthnSessionId,
					name: passKeyName,
					allowSudo,
					isSecondGroup,
				}),
			});

			if (!finishResponse.ok) {
				requestError =
					finishResponse.data?.errors?.[0]?.message ??
					"Passkey registration failed. Please try again.";
				isLoading = false;
				return;
			}

			// Success!
			passKeyName = "";
			allowSudo = false;
			hasStarted = false;
			if (onSuccess) {
				onSuccess();
			} else {
				requestError = "Passkey registered successfully!";
			}
		} finally {
			isLoading = false;
		}
	}

	function handleCancel() {
		hasStarted = false;
		passKeyName = "";
		allowSudo = false;
		requestError = null;
	}
</script>

<Card>
	<CardHeader>
		<CardTitle>Register Passkey</CardTitle>
		<CardDescription>Add a new passkey to your account</CardDescription>
	</CardHeader>
	<CardContent>
		{#if requestError}
			<p class="text-sm text-destructive mb-4">{requestError}</p>
		{/if}

		{#if !hasStarted}
			<div class="space-y-4">
				<div>
					<Label for="passkey-name">Passkey Name</Label>
					<Input
						id="passkey-name"
						placeholder="e.g., My Security Key"
						bind:value={passKeyName}
						disabled={isLoading}
					/>
					<p class="text-sm text-muted-foreground mt-1">
						Give your passkey a descriptive name to help identify it later
					</p>
				</div>

				<div class="flex items-center space-x-2">
					<Checkbox id="allow-sudo" bind:checked={allowSudo} disabled={isLoading} />
					<Label for="allow-sudo" class="cursor-pointer">Allow sudo mode elevation</Label>
				</div>
				<p class="text-sm text-muted-foreground">
					If enabled, this passkey can be used to enter sudo mode for managing your account
				</p>

				<Button
					onclick={() => {
						hasStarted = true;
						handleStartRegistration();
					}}
					disabled={isLoading || !passKeyName.trim()}
					class="w-full"
				>
					{#if isLoading}
						<p>Registering...</p>
					{:else}
						<p>Register Passkey</p>
					{/if}
				</Button>
			</div>
		{:else}
			<div class="space-y-4">
				<p class="text-sm">Follow the prompts on your device to complete passkey registration.</p>
				<Button onclick={handleCancel} disabled={isLoading} variant="outline" class="w-full">
					Cancel
				</Button>
			</div>
		{/if}
	</CardContent>
</Card>

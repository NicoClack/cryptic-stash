<script lang="ts">
	import { resolve } from "$app/paths";
	import { page } from "$app/state";
	import { fetchJson, type JsonResponse } from "$lib/api";
	import PageMain from "$lib/components/PageMain.svelte";
	import { Button } from "$lib/components/ui/button";
	import { Card, CardContent, CardHeader, CardTitle } from "$lib/components/ui/card";
	import { Input } from "$lib/components/ui/input";
	import { Label } from "$lib/components/ui/label";

	interface SignupLinkResponse {
		suggestedName: string;
		expiresAt: string;
	}

	let isLoadingLink = $state(true);
	let isCreating = $state(false);
	let requestError = $state<string | null>(null);
	let successMessage = $state<string | null>(null);

	let username = $state("");
	let password = $state("");
	let confirmPassword = $state("");
	let fileName = $state("");
	let fileContent = $state("");
	let suggestedName = $state("");

	function getErrorMessage(response: JsonResponse): string {
		const firstError = response.data?.errors?.[0];
		if (firstError?.message) {
			return String(firstError.message);
		}
		return `Request failed with status ${response.status}`;
	}

	function normalizeUsername(value: string): string {
		return value.toLowerCase().replace(/[^a-z0-9_-]/g, "");
	}

	function getSignupCode(): string {
		return page.url.searchParams.get("code")?.trim() ?? "";
	}

	function getSignupId(): string {
		return page.params.id ?? "";
	}

	function getAuthHeaders(): HeadersInit {
		const code = getSignupCode();
		if (!code) {
			return {};
		}
		return { Authorization: `Bearer ${code}` };
	}

	function readFileAsBase64(file: File): Promise<string> {
		return new Promise((resolve, reject) => {
			const reader = new FileReader();
			reader.onerror = () => reject(new Error("Failed to read file"));
			reader.onload = () => {
				const result = reader.result;
				if (typeof result !== "string") {
					reject(new Error("File could not be encoded"));
					return;
				}

				const splitIndex = result.indexOf(",");
				if (splitIndex === -1) {
					reject(new Error("File could not be encoded"));
					return;
				}

				resolve(result.slice(splitIndex + 1));
			};
			reader.readAsDataURL(file);
		});
	}

	async function handleFileChange(event: Event) {
		const target = event.currentTarget as HTMLInputElement;
		const selectedFile = target.files?.[0];

		fileContent = "";
		fileName = "";
		if (!selectedFile) {
			return;
		}

		fileName = selectedFile.name;
		try {
			fileContent = await readFileAsBase64(selectedFile);
		} catch {
			requestError = "Could not read selected file.";
		}
	}

	async function loadSignupLink() {
		requestError = null;
		successMessage = null;
		isLoadingLink = true;

		const signupId = getSignupId();
		const code = getSignupCode();
		if (!code) {
			requestError = "Missing signup code. Use the full signup link from your admin.";
			isLoadingLink = false;
			return;
		}

		try {
			const response = await fetchJson(
				fetch,
				`/api/v1/signup-links/${encodeURIComponent(signupId)}`,
				{
					headers: getAuthHeaders(),
				},
			);
			if (!response.ok) {
				requestError = getErrorMessage(response);
				return;
			}

			const data = response.data as SignupLinkResponse;
			suggestedName = data.suggestedName ?? "";
			if (!username && suggestedName) {
				username = suggestedName;
			}
		} finally {
			isLoadingLink = false;
		}
	}

	async function handleSubmit(event: Event) {
		event.preventDefault();
		if (isCreating) return;

		requestError = null;
		successMessage = null;
		const signupId = getSignupId();
		const normalizedUsername = normalizeUsername(username.trim());
		if (!normalizedUsername) {
			requestError = "Username is required.";
			return;
		}
		if (password !== confirmPassword) {
			requestError = "Passwords do not match.";
			return;
		}
		if (!fileContent || !fileName) {
			requestError = "Please select a file to upload.";
			return;
		}

		isCreating = true;
		try {
			const response = await fetchJson(
				fetch,
				`/api/v1/signup-links/${encodeURIComponent(signupId)}/create-user`,
				{
					method: "POST",
					headers: {
						"Content-Type": "application/json",
						...getAuthHeaders(),
					},
					body: JSON.stringify({
						username: normalizedUsername,
						password,
						content: fileContent,
						filename: fileName,
					}),
				},
			);
			if (!response.ok) {
				requestError = getErrorMessage(response);
				return;
			}

			successMessage = "Account created. Please contact your admin to set up your messengers.";
			password = "";
			confirmPassword = "";
		} finally {
			isCreating = false;
		}
	}

	loadSignupLink();
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
				<p class="text-sm text-muted-foreground">Validating signup link...</p>
			{:else}
				<form class="space-y-4" onsubmit={handleSubmit}>
					<Label>
						Username
						<Input
							bind:value={username}
							required
							type="text"
							name="username"
							autocomplete="username"
							maxlength={32}
							oninput={(event) => {
								username = normalizeUsername((event.currentTarget as HTMLInputElement).value);
							}}
						/>
					</Label>

					<Label>
						Password
						<Input
							bind:value={password}
							required
							type="password"
							name="password"
							autocomplete="new-password"
							maxlength={256}
						/>
					</Label>

					<Label>
						Confirm Password
						<Input
							bind:value={confirmPassword}
							required
							type="password"
							name="confirmPassword"
							autocomplete="new-password"
							maxlength={256}
						/>
					</Label>

					<Label>
						Stash File
						<Input required type="file" name="stashFile" oninput={handleFileChange} />
					</Label>

					<Button type="submit" disabled={isCreating}>
						{isCreating ? "Creating..." : "Create Account"}
					</Button>
				</form>
			{/if}
		</CardContent>
	</Card>
</PageMain>

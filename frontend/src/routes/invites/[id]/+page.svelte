<script lang="ts">
	import { resolve } from "$app/paths";
	import { page } from "$app/state";
	import { fetchJson, type JsonResponse } from "$lib/api";
	import PageMain from "$lib/components/PageMain.svelte";
	import { Button } from "$lib/components/ui/button";
	import { Card, CardContent, CardHeader, CardTitle } from "$lib/components/ui/card";
	import { Input } from "$lib/components/ui/input";
	import { Label } from "$lib/components/ui/label";

	interface InviteResponse {
		suggestedName: string;
		expiresAt: string;
	}

	let isLoadingLink = $state(true);
	let isCreating = $state(false);
	let requestError = $state<string | null>(null);
	let successMessage = $state<string | null>(null);

	let username = $state("");
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

	function getInviteCode(): string {
		return page.url.searchParams.get("code")?.trim() ?? "";
	}

	function getInviteId(): string {
		return page.params.id ?? "";
	}

	function getAuthHeaders(): HeadersInit {
		const code = getInviteCode();
		if (!code) {
			return {};
		}
		return { Authorization: `Bearer ${code}` };
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
		const inviteId = getInviteId();
		const normalizedUsername = normalizeUsername(username.trim());
		if (!normalizedUsername) {
			requestError = "Username is required.";
			return;
		}

		isCreating = true;
		try {
			const response = await fetchJson(
				fetch,
				`/api/v1/invites/${encodeURIComponent(inviteId)}/create-user`,
				{
					method: "POST",
					headers: {
						"Content-Type": "application/json",
						...getAuthHeaders(),
					},
					body: JSON.stringify({
						username: normalizedUsername,
					}),
				},
			);
			if (!response.ok) {
				requestError = getErrorMessage(response);
				return;
			}

			successMessage =
				"Account created. Please contact your admin to set up your stash and messengers.";
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

					<Button type="submit" disabled={isCreating}>
						{isCreating ? "Creating..." : "Create Account"}
					</Button>
				</form>
			{/if}
		</CardContent>
	</Card>
</PageMain>

<!-- TODO: un-ai this -->
<script lang="ts">
	import { fetchAdminJson, type InnerResponse, type JsonResponse } from "$lib/api";
	import PageMain from "$lib/components/PageMain.svelte";
	import { Button } from "$lib/components/ui/button";
	import {
		Card,
		CardContent,
		CardDescription,
		CardHeader,
		CardTitle,
	} from "$lib/components/ui/card";
	import { Input } from "$lib/components/ui/input";
	import { Label } from "$lib/components/ui/label";

	interface SignupLink {
		id: string;
		name: string;
		createdAt: string;
		expiresAt: string;
		userId?: string;
		ip: string;
		userAgent: string;
	}
	interface SignupLinksResponse extends InnerResponse {
		signupLinks: SignupLink[];
	}
	interface CreateSignupLinkResponse extends InnerResponse {
		id: string;
		code: string;
		expiresAt: string;
	}

	type CreatedSignupLink = {
		id: string;
		code: string;
		expiresAt: string;
	};

	let isLoading = $state(true);
	let isCreating = $state(false);
	let isRefreshing = $state(false);
	let requestError = $state<string | null>(null);

	let nameFilter = $state("");
	let inviteName = $state("");
	let expiresInHours = $state("24");

	let signupLinks = $state<SignupLink[]>([]);
	let latestInvite = $state<CreatedSignupLink | null>(null);
	let copyState = $state<"idle" | "copied" | "failed">("idle");

	function getErrorMessage(response: JsonResponse): string {
		const firstError = response.data?.errors?.[0];
		if (firstError?.message) {
			return String(firstError.message);
		}
		return `Request failed with status ${response.status}`;
	}

	function formatTime(dateValue: string): string {
		const date = new Date(dateValue);
		return date.toLocaleString();
	}

	function normalizeUsername(value: string): string {
		return value.toLowerCase().replace(/[^a-z0-9_-]/g, "");
	}

	async function copyLatestInviteLink() {
		if (!latestInvite) return;

		// TODO: use URL safe base64?
		const signupUrl = new URL(
			`/signup/${latestInvite.id}/?code=${encodeURIComponent(latestInvite.code)}`,
			document.baseURI,
		).toString();
		try {
			await navigator.clipboard.writeText(signupUrl);
			copyState = "copied";
		} catch {
			copyState = "failed";
		}
	}

	async function loadSignupLinks() {
		isRefreshing = true;
		requestError = null;

		try {
			const query = normalizeUsername(nameFilter.trim());
			const suffix = query ? `?name=${encodeURIComponent(query)}` : "";
			const response = await fetchAdminJson(fetch, `/api/v1/admin/signup-links/${suffix}`);
			if (!response.ok) {
				requestError = getErrorMessage(response);
				return;
			}

			const data = response.data as SignupLinksResponse;
			signupLinks = data.signupLinks ?? [];
		} finally {
			isLoading = false;
			isRefreshing = false;
		}
	}

	async function handleCreate(event: Event) {
		event.preventDefault();
		if (isCreating) return;

		isCreating = true;
		requestError = null;

		try {
			const parsedHours = Number(expiresInHours);
			const expiresInSeconds =
				isNaN(parsedHours) || parsedHours < 0 ? null : Math.floor(parsedHours * 3600);

			const response = await fetchAdminJson(fetch, "/api/v1/admin/signup-links/create", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({
					name: normalizeUsername(inviteName.trim()),
					expiresIn: expiresInSeconds,
				}),
			});

			if (!response.ok) {
				requestError = getErrorMessage(response);
				return;
			}

			const data = response.data as CreateSignupLinkResponse;
			latestInvite = {
				id: data.id,
				code: data.code,
				expiresAt: data.expiresAt,
			};
			copyState = "idle";
			inviteName = "";
			await loadSignupLinks();
		} finally {
			isCreating = false;
		}
	}

	loadSignupLinks();
</script>

<PageMain class="max-w-5xl">
	<div class="space-y-6">
		<div class="space-y-2">
			<h1 class="text-3xl text-balance font-semibold tracking-tight">Signup Links</h1>
		</div>

		{#if requestError}
			<div
				class="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive"
			>
				{requestError}
			</div>
		{/if}

		<Card>
			<CardHeader>
				<CardTitle>Create Signup Link</CardTitle>
				<CardDescription>
					The generated code is only returned once. Copy it immediately.
				</CardDescription>
			</CardHeader>
			<CardContent>
				<form class="flex flex-col gap-4" onsubmit={handleCreate}>
					<Label>
						Suggested username (optional)
						<Input
							bind:value={inviteName}
							name="name"
							maxlength={32}
							oninput={(event) => {
								inviteName = normalizeUsername((event.currentTarget as HTMLInputElement).value);
							}}
						/>
					</Label>
					<Label>
						Expires In (hours)
						<Input bind:value={expiresInHours} required type="number" min="0" step="1" />
					</Label>
					<div class="flex items-end">
						<Button type="submit" disabled={isCreating || isRefreshing}>
							{isCreating ? "Creating..." : "Create"}
						</Button>
					</div>
				</form>

				{#if latestInvite}
					<div class="mt-4 rounded-md border border-border bg-muted/40 p-3 text-sm space-y-2">
						<div>
							<Button type="button" variant="outline" onclick={copyLatestInviteLink}>
								Copy signup link
							</Button>
							{#if copyState === "copied"}
								<span class="ml-2 text-muted-foreground">Copied</span>
							{:else if copyState === "failed"}
								<span class="ml-2 text-destructive">Could not copy</span>
							{/if}
						</div>
						<div>
							<span class="text-muted-foreground">Expires:</span>
							{formatTime(latestInvite.expiresAt)}
						</div>
					</div>
				{/if}
			</CardContent>
		</Card>

		<Card>
			<CardHeader>
				<CardTitle>Existing Signup Links</CardTitle>
				<CardDescription>Filter by name prefix and review status for each link.</CardDescription>
			</CardHeader>
			<CardContent class="space-y-4">
				<form
					class="flex flex-col gap-3 sm:flex-row"
					onsubmit={(event) => {
						event.preventDefault();
						loadSignupLinks();
					}}
				>
					<Input
						bind:value={nameFilter}
						name="nameFilter"
						placeholder="Filter by name prefix"
						oninput={(event) => {
							nameFilter = normalizeUsername((event.currentTarget as HTMLInputElement).value);
						}}
					/>
					<!-- TODO: remove button, search on type -->
					<Button type="submit" variant="outline" disabled={isRefreshing}>
						{isRefreshing ? "Refreshing..." : "Apply Filter"}
					</Button>
				</form>

				{#if isLoading}
					<p class="text-sm text-muted-foreground">Loading signup links...</p>
				{:else if signupLinks.length === 0}
					<p class="text-sm text-muted-foreground">No signup links found.</p>
				{:else}
					<div class="space-y-3">
						{#each signupLinks as signupLink (signupLink.id)}
							<div class="rounded-md border border-border p-3 space-y-2">
								<div class="flex flex-wrap items-center justify-between gap-2">
									<div class="font-medium">{signupLink.name || "(no name)"}</div>
									<span class="text-xs text-muted-foreground">{signupLink.id}</span>
								</div>
								<div class="grid gap-1 text-sm text-muted-foreground">
									<div>Created: {formatTime(signupLink.createdAt)}</div>
									<div>Expires: {formatTime(signupLink.expiresAt)}</div>
									<div>User ID: {signupLink.userId || "Unused"}</div>
									<div>IP: {signupLink.ip || "-"}</div>
									<div class="break-all">User Agent: {signupLink.userAgent || "-"}</div>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</CardContent>
		</Card>
	</div>
</PageMain>

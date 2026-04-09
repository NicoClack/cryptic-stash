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
	import { Textarea } from "$lib/components/ui/textarea";

	interface Invite {
		id: string;
		email: string;
		createdAt: string;
		expiresAt: string;
		userId?: string;
		ip: string;
		userAgent: string;
	}
	interface InviteLinksResponse extends InnerResponse {
		inviteLinks?: Invite[];
	}
	interface CreateInviteResponse extends InnerResponse {
		id: string;
		code: string;
		expiresAt: string;
	}

	const DEFAULT_INVITE_MESSAGE =
		"<YOUR NAME HERE> invited you to Cryptic Stash, an end-to-end encrypted storage site for securely storing 2FA recovery codes in case you lose your devices.";

	type CreatedInvite = {
		id: string;
		code: string;
		expiresAt: string;
	};

	let isLoading = $state(true);
	let isCreating = $state(false);
	let isRefreshing = $state(false);
	let requestError = $state<string | null>(null);

	let emailFilter = $state("");
	let email = $state("");
	let message = $state(DEFAULT_INVITE_MESSAGE);
	let expiresInHours = $state("24");

	let inviteLinks = $state<Invite[]>([]);
	let latestInvite = $state<CreatedInvite | null>(null);
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

	function normalizeEmail(value: string): string {
		return value.trim().toLowerCase();
	}

	async function copyLatestInviteLink() {
		if (!latestInvite) return;

		// TODO: use URL safe base64?
		const inviteUrl = new URL(
			`/invites/${latestInvite.id}/?code=${encodeURIComponent(latestInvite.code)}`,
			document.baseURI,
		).toString();
		try {
			await navigator.clipboard.writeText(inviteUrl);
			copyState = "copied";
		} catch {
			copyState = "failed";
		}
	}

	async function loadInviteLinks() {
		isRefreshing = true;
		requestError = null;

		try {
			const query = normalizeEmail(emailFilter);
			const suffix = query ? `?email=${encodeURIComponent(query)}` : "";
			const response = await fetchAdminJson(fetch, `/api/v1/admin/invites/${suffix}`);
			if (!response.ok) {
				requestError = getErrorMessage(response);
				return;
			}

			const data = response.data as InviteLinksResponse;
			inviteLinks = data.inviteLinks ?? [];
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
			const normalizedEmail = normalizeEmail(email);
			if (!normalizedEmail) {
				requestError = "Email is required";
				return;
			}

			const trimmedMessage = message.trim();
			if (!trimmedMessage) {
				requestError = "Message is required";
				return;
			}

			const parsedHours = Number(expiresInHours);
			const expiresInSeconds =
				isNaN(parsedHours) || parsedHours < 0 ? null : Math.floor(parsedHours * 3600);

			const response = await fetchAdminJson(fetch, "/api/v1/admin/invites/create", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({
					email: normalizedEmail,
					inviteMessage: trimmedMessage,
					expiresIn: expiresInSeconds,
				}),
			});

			if (!response.ok) {
				requestError = getErrorMessage(response);
				return;
			}

			const data = response.data as CreateInviteResponse;
			latestInvite = {
				id: data.id,
				code: data.code,
				expiresAt: data.expiresAt,
			};
			copyState = "idle";
			email = "";
			message = DEFAULT_INVITE_MESSAGE;
			await loadInviteLinks();
		} finally {
			isCreating = false;
		}
	}

	loadInviteLinks();
</script>

<PageMain class="max-w-5xl">
	<div class="space-y-6">
		<div class="space-y-2">
			<h1 class="text-3xl text-balance font-semibold tracking-tight">Invites</h1>
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
				<CardTitle>Create Invite</CardTitle>
				<CardDescription>
					The generated code is only returned once and the invite email is sent immediately.
				</CardDescription>
			</CardHeader>
			<CardContent>
				<form class="flex flex-col gap-4" onsubmit={handleCreate}>
					<Label>
						Email
						<Input
							bind:value={email}
							name="email"
							type="email"
							required
							maxlength={128}
							oninput={(event) => {
								email = normalizeEmail((event.currentTarget as HTMLInputElement).value);
							}}
						/>
					</Label>
					<Label>
						Message
						<Textarea bind:value={message} name="inviteMessage" required maxlength={500} rows={4} />
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
								Copy invite link
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
				<CardTitle>Existing Invites</CardTitle>
				<CardDescription>Filter by name prefix and review status for each link.</CardDescription>
			</CardHeader>
			<CardContent class="space-y-4">
				<form
					class="flex flex-col gap-3 sm:flex-row"
					onsubmit={(event) => {
						event.preventDefault();
						loadInviteLinks();
					}}
				>
					<Input
						bind:value={emailFilter}
						name="emailFilter"
						placeholder="Filter by email prefix"
						oninput={(event) => {
							emailFilter = normalizeEmail((event.currentTarget as HTMLInputElement).value);
						}}
					/>
					<!-- TODO: remove button, search on type -->
					<Button type="submit" variant="outline" disabled={isRefreshing}>
						{isRefreshing ? "Refreshing..." : "Apply Filter"}
					</Button>
				</form>

				{#if isLoading}
					<p class="text-sm text-muted-foreground">Loading invites...</p>
				{:else if inviteLinks.length === 0}
					<p class="text-sm text-muted-foreground">No invites found.</p>
				{:else}
					<div class="space-y-3">
						{#each inviteLinks as invite (invite.id)}
							<div class="rounded-md border border-border p-3 space-y-2">
								<div class="flex flex-wrap items-center justify-between gap-2">
									<div class="font-medium">{invite.email}</div>
									<span class="text-xs text-muted-foreground">{invite.id}</span>
								</div>
								<div class="grid gap-1 text-sm text-muted-foreground">
									<div>Created: {formatTime(invite.createdAt)}</div>
									<div>Expires: {formatTime(invite.expiresAt)}</div>
									<div>User ID: {invite.userId || "Unused"}</div>
									<div>IP: {invite.ip || "-"}</div>
									<div class="break-all">User Agent: {invite.userAgent || "-"}</div>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</CardContent>
		</Card>
	</div>
</PageMain>

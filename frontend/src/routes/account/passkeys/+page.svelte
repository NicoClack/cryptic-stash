<script lang="ts">
	import { resolve } from "$app/paths";
	import { fetchUserJson, type ApiErrorDetail } from "$lib/api";
	import PageMain from "$lib/components/PageMain.svelte";
	import RegisterPasskey from "$lib/components/RegisterPasskey.svelte";
	import { Button } from "$lib/components/ui/button";
	import {
		Card,
		CardContent,
		CardDescription,
		CardHeader,
		CardTitle,
	} from "$lib/components/ui/card";
	import { onMount } from "svelte";

	interface PasskeyInfo {
		id: string;
		name: string;
		allowSudo: boolean;
		isSessionFirst: boolean;
		isSessionSecond: boolean;
	}

	interface PasskeyListResponse {
		errors: ApiErrorDetail[];
		firstGroupPasskeys: PasskeyInfo[];
		secondGroupPasskeys: PasskeyInfo[];
	}

	const passkeysUrl = "/api/v1/self/passkeys/";

	let firstGroupPasskeys = $state<PasskeyInfo[]>([]);
	let secondGroupPasskeys = $state<PasskeyInfo[]>([]);
	let isLoading = $state(true);
	let requestError = $state<string | null>(null);
	let busyPasskeyID = $state<string | null>(null);
	let isDisablingTwoGroupAuth = $state(false);
	let registrationDialog = $state<HTMLDialogElement | null>(null);
	let registrationGroup = $state<boolean | null>(null);

	function getResponseError(data: { errors?: ApiErrorDetail[] }): string {
		return data.errors?.[0]?.message ?? "The request failed. Please try again.";
	}

	async function loadPasskeys() {
		isLoading = true;
		requestError = null;

		try {
			const response = await fetchUserJson(fetch, passkeysUrl);
			if (!response.ok) {
				requestError = getResponseError(response.data);
				return;
			}

			const data = response.data as PasskeyListResponse;
			firstGroupPasskeys = data.firstGroupPasskeys;
			secondGroupPasskeys = data.secondGroupPasskeys;
		} catch {
			requestError = "Unable to load your passkeys. Please try again.";
		} finally {
			isLoading = false;
		}
	}

	async function updatePasskey(
		passkey: PasskeyInfo,
		path: string,
		body?: Record<string, unknown>,
	): Promise<boolean> {
		busyPasskeyID = passkey.id;
		requestError = null;

		try {
			const response = await fetchUserJson(fetch, `${passkeysUrl}${passkey.id}/${path}/`, {
				method: "POST",
				headers: body ? { "Content-Type": "application/json" } : undefined,
				body: body ? JSON.stringify(body) : undefined,
			});
			if (!response.ok) {
				requestError = getResponseError(response.data);
				return false;
			}
			await loadPasskeys();
			return true;
		} catch {
			requestError = "Unable to update this passkey. Please try again.";
			return false;
		} finally {
			busyPasskeyID = null;
		}
	}

	async function renamePasskey(passkey: PasskeyInfo) {
		const name = window.prompt("Passkey name", passkey.name)?.trim();
		if (!name || name === passkey.name) return;
		await updatePasskey(passkey, "rename", { name });
	}

	async function deletePasskey(passkey: PasskeyInfo) {
		if (passkey.isSessionFirst || passkey.isSessionSecond) return;
		if (!window.confirm(`Delete the passkey "${passkey.name}"?`)) return;
		await updatePasskey(passkey, "delete");
	}

	async function toggleSudo(passkey: PasskeyInfo) {
		await updatePasskey(passkey, "update-sudo", { allowSudo: !passkey.allowSudo });
	}

	async function movePasskey(passkey: PasskeyInfo) {
		if (!window.confirm(`Move "${passkey.name}" to the other group?`)) return;
		const isSecondGroup = !secondGroupPasskeys.some((item) => item.id === passkey.id);
		await updatePasskey(passkey, "move-group", { isSecondGroup });
	}

	async function disableTwoGroupAuth() {
		if (
			!window.confirm(
				"Are you sure you want to disable two-group authentication? This will allow you to use a single sudo passkey twice to enter sudo mode",
			)
		)
			return;
		isDisablingTwoGroupAuth = true;
		requestError = null;
		try {
			const response = await fetchUserJson(fetch, `${passkeysUrl}disable-two-group-auth/`, {
				method: "POST",
			});
			if (!response.ok) {
				requestError = getResponseError(response.data);
				return;
			}
			await loadPasskeys();
		} catch {
			requestError = "Unable to disable two-group authentication. Please try again.";
		} finally {
			isDisablingTwoGroupAuth = false;
		}
	}

	function handleSuccess() {
		void loadPasskeys();
		registrationDialog?.close();
		registrationGroup = null;
	}

	function openRegistration(isSecondGroup: boolean) {
		registrationGroup = isSecondGroup;
		requestAnimationFrame(() => registrationDialog?.showModal());
	}

	onMount(() => {
		void loadPasskeys();
	});
</script>

<PageMain>
	<div class="w-full max-w-3xl space-y-6">
		<div>
			<h1 class="text-2xl font-semibold">Passkeys</h1>
			<p class="text-muted-foreground">Manage the passkeys that protect your account.</p>
		</div>

		{#if requestError}
			<p class="text-sm text-destructive">{requestError}</p>
		{/if}

		<Card>
			<CardHeader>
				<CardTitle>Your passkeys</CardTitle>
				<CardDescription>Passkeys are grouped to support stronger account recovery.</CardDescription
				>
			</CardHeader>
			<CardContent class="space-y-6">
				{#if isLoading}
					<p class="text-sm text-muted-foreground">Loading passkeys...</p>
				{:else if firstGroupPasskeys.length === 0 && secondGroupPasskeys.length === 0}
					<p class="text-sm text-muted-foreground">No passkeys registered yet.</p>
				{:else}
					{#each [{ title: "First group", items: firstGroupPasskeys }, { title: "Second group", items: secondGroupPasskeys }] as group (group.title)}
						<section class="space-y-3">
							<h2 class="font-medium">{group.title}</h2>
							{#if group.items.length === 0}
								{#if group.title === "First group"}
									<p class="text-sm text-muted-foreground">
										No first-group passkeys have been registered yet.
									</p>
								{:else}
									<p class="text-sm text-muted-foreground">
										Two keys to attack. One key to defend.
									</p>
									<p class="text-sm text-muted-foreground">
										Increase your account security by registering or moving a passkey to the second
										group. You will then need one passkey from each group in order to perform
										sensitive actions such as deleting a stash. While this technically isn't an
										extra factor, it enables you to take advantage of the real-world strengths of
										different types of passkeys:
									</p>
									<ul
										class="list-disc space-y-2 pl-5 text-sm text-muted-foreground marker:text-foreground/60"
									>
										<li class="pl-1">
											Major: passkeys stored on a hardware key can't be copied by malware or
											manipulation. Also it's difficult for malware to silently use a hardware key
											to log in, as it requires a physical tap.
										</li>
										<li class="pl-1">
											Medium: hardware keys are less likely to be stolen than a device logged into
											your password manager, as they are, or at least appear to be, less valuable.
										</li>
										<li class="pl-1">
											Medium: hardware keys have a much lower attack surface area than a password
											manager that runs alongside an OS, browser, extensions and other apps.
										</li>
										<li class="pl-1">
											Medium: synced passkey providers usually support biometrics, which can't be
											shoulder-surfed.
										</li>
									</ul>
									<p class="text-sm text-muted-foreground">
										Because each of these two types are compromised in different scenarios, you
										should put each type in a separate group. That way two scenarios have to occur.
										You can still block a download attempt with any single passkey.
									</p>
									<p class="text-sm text-muted-foreground">
										Alternatively, if you only want to use a single passkey, you can disable sudo
										access for your synced passkeys and use a single group. Synced passkeys are
										overall less secure, but they're more accessible if you need to block a download
										attempt. Then to make sensitive changes, you'll need your hardware key.
									</p>
								{/if}
							{:else}
								{#each group.items as passkey (passkey.id)}
									<div class="space-y-3 rounded-md border p-4">
										<div class="flex flex-col flex-wrap items-start justify-between gap-3">
											<div>
												<p class="font-medium">{passkey.name}</p>
												<p class="text-sm text-muted-foreground">
													{passkey.allowSudo ? "Sudo" : "Non-sudo"}
													{#if passkey.isSessionFirst && passkey.isSessionSecond}
														· Was used to log in and elevate your session
													{:else if passkey.isSessionFirst}
														· Was used to log in
													{:else if passkey.isSessionSecond}
														· Was used to elevate your session
													{/if}
												</p>
											</div>
											<div class="flex flex-wrap gap-2">
												<Button
													variant="outline"
													onclick={() => renamePasskey(passkey)}
													disabled={busyPasskeyID === passkey.id}>Rename</Button
												>
												<Button
													variant="outline"
													onclick={() => toggleSudo(passkey)}
													disabled={busyPasskeyID === passkey.id}
													>{passkey.allowSudo ? "Disable sudo" : "Enable sudo"}</Button
												>
												<Button
													variant="outline"
													onclick={() => movePasskey(passkey)}
													disabled={busyPasskeyID === passkey.id}>Move group</Button
												>
												<Button
													variant="destructive"
													onclick={() => deletePasskey(passkey)}
													disabled={busyPasskeyID === passkey.id ||
														passkey.isSessionFirst ||
														passkey.isSessionSecond}>Delete</Button
												>
											</div>
										</div>
										{#if passkey.isSessionFirst || passkey.isSessionSecond}
											<p class="text-xs text-muted-foreground">
												This passkey is currently used by your session and cannot be deleted. If you
												still want to delete it, please log in with a different passkey first.
											</p>
										{/if}
									</div>
								{/each}
							{/if}
							<Button
								variant="outline"
								onclick={() => openRegistration(group.title === "Second group")}
							>
								Add a new passkey
							</Button>
						</section>
					{/each}
				{/if}

				{#if secondGroupPasskeys.length > 0}
					<Button
						variant="outline"
						onclick={disableTwoGroupAuth}
						disabled={isDisablingTwoGroupAuth}
					>
						Disable two-group authentication
					</Button>
				{/if}
			</CardContent>
		</Card>

		{#if registrationGroup !== null}
			<dialog
				bind:this={registrationDialog}
				class="m-auto w-[calc(100%-2rem)] max-w-lg rounded-lg border bg-background p-0 text-foreground shadow-lg backdrop:bg-black/50"
				onclose={() => (registrationGroup = null)}
			>
				{#key registrationGroup}
					<RegisterPasskey isSecondGroup={registrationGroup} onSuccess={handleSuccess} />
				{/key}
			</dialog>
		{/if}

		<div>
			<Button href={resolve("/")} variant="ghost" class="w-full">Back</Button>
		</div>
	</div>
</PageMain>

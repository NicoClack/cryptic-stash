<script lang="ts">
	import { adminAuth } from "$lib/admin/AdminAuth.svelte";
	import { fetchJson } from "$lib/api";
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
	let isLoading = $state(false);

	let password = $state("");
	let totpCode = $state("");

	async function handleSubmit(event: Event) {
		event.preventDefault();
		if (isLoading) return;
		isLoading = true;

		try {
			const response = await fetchJson(fetch, "/api/v1/admin/login/", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({
					password,
					totpCode: totpCode.replace(/\s/g, ""),
				}),
			});
			if (response.redirecting || !response.ok) {
				isLoading = false;
				response.throwForStatus();
				return;
			}
			adminAuth.login(response.data.adminCode, response.data.adminUserId);
		} finally {
			isLoading = false;
		}
	}
</script>

<main>
	<Card>
		<CardHeader>
			<CardTitle>Admin Login</CardTitle>
			<CardDescription>Sign in with your admin password and current 2FA code.</CardDescription>
		</CardHeader>
		<CardContent>
			<form class="space-y-4" onsubmit={handleSubmit}>
				<Label>
					Username
					<Input
						required
						disabled
						type="text"
						name="username"
						autocomplete="username"
						value="admin"
					/>
				</Label>
				<Label>
					Password
					<Input
						bind:value={password}
						required
						type="password"
						name="password"
						autocomplete="current-password"
						maxlength={256}
					/>
				</Label>
				<Label>
					2FA Code
					<Input
						bind:value={totpCode}
						required
						type="text"
						id="otp"
						name="otp"
						inputmode="numeric"
						pattern="[0-9\s]*"
						autocomplete="one-time-code"
					/>
				</Label>
				<Button type="submit" disabled={isLoading}>Login</Button>
			</form>
		</CardContent>
	</Card>
</main>

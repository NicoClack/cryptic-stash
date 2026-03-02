<script lang="ts">
	import { adminAuth } from "$lib/admin/AdminAuth.svelte";
	import { fetchJson } from "$lib/api";
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
	<h3>Admin Login</h3>
	<form onsubmit={handleSubmit}>
		<label>
			Username:
			<input required disabled type="text" name="username" autocomplete="username" value="admin" />
		</label> <br />
		<label>
			Password:
			<input
				bind:value={password}
				required
				type="password"
				name="password"
				autocomplete="current-password"
				maxlength="256"
			/>
		</label> <br />
		<label>
			2FA Code:
			<input
				bind:value={totpCode}
				required
				type="text"
				id="otp"
				name="otp"
				inputmode="numeric"
				pattern="[0-9\s]*"
				autocomplete="one-time-code"
			/>
		</label> <br />
		<button type="submit" disabled={isLoading}>Login</button>
	</form>
</main>

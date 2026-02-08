<script lang="ts">
	import type { AdminEnvVars } from "$lib/admin/setup/setup/setup/setup";
	import { fetchJson } from "$lib/api";

	const RANDOM_PASSWORD_LENGTH = 128;

	const {
		onComplete,
	}: {
		onComplete: (adminEnvVars: AdminEnvVars) => unknown;
	} = $props();

	let isLoading = $state(false);

	let password = $state("");
	let confirmPassword = $state("");

	async function handleSubmit(event: Event) {
		event.preventDefault();
		if (isLoading) return;
		isLoading = true;

		if (password !== confirmPassword) {
			isLoading = false;
			alert("Passwords do not match");
			return;
		}

		const response = await fetchJson(fetch, "/api/v1/setup/generate-admin-env-vars/", {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ password }),
		});
		if (response.redirecting || !response.ok) {
			isLoading = false;
			response.throwForStatus();
			return;
		}
		await onComplete(response.data);
		isLoading = false;
	}

	function handleRandomPassword() {
		// TODO: is this correct?
		const generatedPassword = crypto
			.getRandomValues(new Uint8Array(RANDOM_PASSWORD_LENGTH))
			.reduce((str, byte) => str + ("0" + byte.toString(16)).slice(-2), "");

		password = generatedPassword;
		confirmPassword = generatedPassword;
	}
</script>

<main>
	<h3>Step 1 of 4: Admin Password</h3>
	<form onsubmit={handleSubmit}>
		<label>
			Username:
			<input required disabled type="text" name="username" autocomplete="username" value="admin" />
		</label> <br />
		<div>
			<label>
				Password:
				<input
					bind:value={password}
					required
					type="password"
					name="password"
					autocomplete="new-password"
					maxlength="256"
				/>
			</label>
			<button type="button" onclick={handleRandomPassword}>
				Random {RANDOM_PASSWORD_LENGTH} character password
			</button>
		</div>
		<label>
			Confirm Password:
			<input
				bind:value={confirmPassword}
				required
				type="password"
				name="confirm-password"
				autocomplete="new-password"
				maxlength="256"
			/>
		</label> <br />
		<p>
			Note: We recommend using the random password button and storing it in your password manager.
			This allows you to weaken the hashing for admin passwords, reducing server load. If you must
			use a memorable password, use the correct horse battery staple method and set your admin
			hashing env vars to match the recommendations for stash passwords.
		</p>
		<button type="submit" disabled={isLoading}>Next</button>
	</form>
</main>

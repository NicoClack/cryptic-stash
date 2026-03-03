<script lang="ts">
	import { Card, CardDescription, CardHeader, CardTitle } from "$lib/components/ui/card";
</script>

<main>
	<Card>
		<CardHeader>
			<CardTitle class="text-3xl">Welcome to Cryptic Stash</CardTitle>
			<CardDescription class="text-base">
				Securely store and recover your 2FA recovery codes in case you lose access to your devices.
				If an attacker tries to log in, you'll be notified and can block it.
			</CardDescription>
		</CardHeader>
	</Card>
</main>

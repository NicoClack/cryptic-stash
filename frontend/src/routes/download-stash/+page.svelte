<script lang="ts">
	import { resolve } from "$app/paths";
	import { Card, CardDescription, CardHeader, CardTitle } from "$lib/components/ui/card";
</script>

<main>
	<h1 class="text-center text-3xl">
		<a href={resolve("/")}>Cryptic Stash</a>
	</h1>

	<Card>
		<CardHeader>
			<CardTitle>Download your stash</CardTitle>
			<CardDescription>
				This page is coming soon. You'll be able to request and download your encrypted stash here.
			</CardDescription>
		</CardHeader>
	</Card>
</main>

<script lang="ts">
	import { resolve } from "$app/paths";
	import { Card, CardDescription, CardHeader, CardTitle } from "$lib/components/ui/card";
</script>

<main class="mx-auto w-full max-w-3xl space-y-6 px-6 py-10">
	<h1 class="text-center text-3xl">
		<a
			href={resolve("/")}
			class="text-primary underline-offset-4 outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
			>Cryptic Stash</a
		>
	</h1>

	<Card>
		<CardHeader>
			<CardTitle>Freeze your account</CardTitle>
			<CardDescription>
				This page is coming soon. You'll be able to temporarily block stash download requests here.
			</CardDescription>
		</CardHeader>
	</Card>
</main>

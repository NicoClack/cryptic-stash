<script lang="ts">
	import { resolve } from "$app/paths";
	import PageMain from "$lib/components/PageMain.svelte";
	import { Card, CardHeader, CardTitle } from "$lib/components/ui/card";
</script>

<PageMain>
	<h1 class="text-3xl text-balance font-semibold tracking-tight">Cryptic Stash Admin</h1>

	<Card>
		<CardHeader>
			<CardTitle>
				<a href={resolve("/admin/signup-links")}>Signup Links</a></CardTitle
			>
		</CardHeader>
	</Card>
</PageMain>

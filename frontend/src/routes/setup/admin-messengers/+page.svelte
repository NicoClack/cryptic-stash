<script lang="ts">
	import { getAdminAuth } from "$lib/admin/AdminAuth.svelte";
	import MessengersList from "$lib/admin/messengers/MessengersList.svelte";

	const { userID } = getAdminAuth();
</script>

<div class="mx-auto max-w-3xl py-12 px-6">
	<header class="mb-10">
		<h1 class="text-4xl font-extrabold tracking-tight lg:text-5xl">Admin Messenger Setup</h1>
		<p class="mt-4 text-xl text-muted-foreground">
			To ensure we can alert you if there's an error, a user session is created or a few other
			things happen, please set up at least one messenger.
		</p>
	</header>

	{#if userID}
		<div class="space-y-8">
			<MessengersList {userID} />
		</div>
	{/if}
</div>

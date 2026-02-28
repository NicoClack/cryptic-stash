<script lang="ts">
	import { fetchAdminJson } from "$lib/api";
	import Messenger from "./Messenger.svelte";

	const { userID }: { userID: string } = $props();

	let messengersPromise = $derived(
		fetchAdminJson(fetch, `/api/v1/admin/users/${userID}/messengers/`, {
			throwForStatus: true,
		}),
	);
</script>

{#await messengersPromise then response}
	{#each Object.entries<any>(response.data.messengers) as [id, messenger]}
		<Messenger
			{id}
			{userID}
			name={messenger.name}
			enabled={messenger.enabled}
			options={messenger.options}
			optionsSchema={messenger.optionsSchema}
		/>
	{/each}
{/await}

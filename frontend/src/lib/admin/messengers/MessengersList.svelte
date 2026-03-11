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
	{#each response.data.messengers as messenger (messenger.versionedType)}
		<Messenger
			versionedType={messenger.versionedType}
			{userID}
			name={messenger.name}
			enabled={messenger.enabled}
			options={messenger.options}
			optionsSchema={messenger.optionsSchema}
		/>
	{/each}
{/await}

<script lang="ts">
	import { fetchJson } from "$lib/api";
	import Messenger from "./Messenger.svelte";

	const { userID }: { userID: string } = $props();

	let messengersPromise = $derived(
		fetchJson(fetch, `/admin/users/${userID}/messengers`, {
			throwForStatus: true,
		}),
	);
</script>

{#await messengersPromise then response}
	{#each response.data.messengers as messenger}
		<Messenger
			name={messenger.name}
			enabled={messenger.enabled}
			options={messenger.options}
			optionsSchema={messenger.optionsSchema}
		/>
	{/each}
{/await}

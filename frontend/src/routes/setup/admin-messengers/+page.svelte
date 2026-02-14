<script lang="ts">
	import { getAdminAuth } from "$lib/admin/AdminAuth.svelte";
	import MessengersList from "$lib/admin/messengers/MessengersList.svelte";

	const { userID } = getAdminAuth();
</script>

<h2>Admin Messenger Setup</h2>
<p>
	To ensure we can alert you if there's an error, a user session is created or a few other things
	happen, please set up at least one messenger.
</p>

{#if userID}
	<MessengersList {userID} />
{/if}

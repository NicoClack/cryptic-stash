<script lang="ts">
	import { adminAuth } from "$lib/admin/AdminAuth.svelte";
	import MessengersList from "$lib/admin/messengers/MessengersList.svelte";
	import PageMain from "$lib/components/PageMain.svelte";
	import {
		Card,
		CardContent,
		CardDescription,
		CardHeader,
		CardTitle,
	} from "$lib/components/ui/card";
</script>

<PageMain>
	<Card>
		<CardHeader>
			<CardTitle>Admin Messenger Setup</CardTitle>
			<CardDescription>
				To ensure we can alert you if there's an error, a user session is created or a few other
				things happen, please set up at least one messenger for your admin user.
			</CardDescription>
		</CardHeader>
		<CardContent>
			{#if adminAuth.userID}
				<div class="space-y-6">
					<MessengersList userID={adminAuth.userID} />
				</div>
			{/if}
		</CardContent>
	</Card>
</PageMain>

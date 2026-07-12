<script lang="ts">
	import { resolve } from "$app/paths";
	import PageMain from "$lib/components/PageMain.svelte";
	import RegisterPasskey from "$lib/components/RegisterPasskey.svelte";
	import { Button } from "$lib/components/ui/button";

	function handleSuccess() {
		// Redirect to passkeys list (or main account page once implemented)
		window.location.href = resolve("/");
	}
</script>

<PageMain>
	<div class="max-w-md">
		<RegisterPasskey onSuccess={handleSuccess} />

		<div class="mt-4">
			<!-- TODO: partly reset the component rather than going home, don't reset the fields -->
			<Button href={resolve("/")} variant="ghost" class="w-full">Back</Button>
		</div>
	</div>
</PageMain>

<script lang="ts">
	import { fetchAdminJson } from "$lib/api";
	import { Button } from "$lib/components/ui/button";
	import CardContent from "$lib/components/ui/card/card-content.svelte";
	import CardDescription from "$lib/components/ui/card/card-description.svelte";
	import CardHeader from "$lib/components/ui/card/card-header.svelte";
	import CardTitle from "$lib/components/ui/card/card-title.svelte";
	import Card from "$lib/components/ui/card/card.svelte";
	import JsonForm from "$lib/form/JsonForm.svelte";
	import { cn } from "$lib/utils";

	let {
		versionedType,
		userID,
		name,
		enabled = $bindable(false),
		options,
		optionsSchema,
	}: {
		versionedType: string;
		userID: string;
		name: string;
		enabled: boolean;
		options: Record<string, any>;
		optionsSchema: Record<string, any>;
	} = $props();

	let isEnabling = $state(false);
	let isDisabling = $state(false);

	async function handleSubmit(value: unknown) {
		isEnabling = true;
		try {
			await fetchAdminJson(fetch, `/api/v1/admin/users/${userID}/messengers/enable/`, {
				method: "POST",
				body: JSON.stringify({
					versionedType,
					options: value,
				}),
				throwForStatus: true,
			});
			enabled = true;
		} finally {
			isEnabling = false;
		}
	}

	async function handleDisable() {
		if (isDisabling) return;

		isDisabling = true;
		try {
			await fetchAdminJson(fetch, `/api/v1/admin/users/${userID}/messengers/disable/`, {
				method: "POST",
				body: JSON.stringify({
					versionedType,
				}),
				throwForStatus: true,
			});
			enabled = false;
		} finally {
			isDisabling = false;
		}
	}
</script>

<Card>
	<CardHeader>
		<CardTitle>{name}</CardTitle>
		<CardDescription>
			<div class="flex items-center justify-between gap-2">
				<span
					class={cn(
						"inline-flex items-center rounded-full px-2 py-1 text-xs font-medium ring-1 ring-inset bg-muted text-muted-foreground ring-border",
					)}
				>
					{enabled ? "Enabled" : "Disabled"}
				</span>
				{#if enabled}
					<Button
						type="button"
						variant="outline"
						onclick={handleDisable}
						disabled={isDisabling || isEnabling}
					>
						Disable
					</Button>
				{/if}
			</div>
		</CardDescription>
	</CardHeader>
	<CardContent>
		{#if optionsSchema}
			<!-- TODO: is there a way to remove this #key? It's needed so that the submitLabel updates on submit -->
			{#key enabled}
				<JsonForm
					schema={optionsSchema}
					initialValue={options}
					onSubmit={handleSubmit}
					isDisabled={isEnabling || isDisabling}
					submitLabel={enabled ? "Update" : "Enable"}
				/>
			{/key}
		{/if}
	</CardContent>
</Card>

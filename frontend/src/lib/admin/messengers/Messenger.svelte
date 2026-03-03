<script lang="ts">
	import { fetchAdminJson } from "$lib/api";
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
		enabled,
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

	let isSubmitting = $state(false);

	async function handleSubmit(value: unknown) {
		isSubmitting = true;
		try {
			await fetchAdminJson(fetch, `/api/v1/admin/users/${userID}/messengers/enable/`, {
				method: "POST",
				body: JSON.stringify({
					versionedType,
					options: value,
				}),
				throwForStatus: true,
			});
		} finally {
			isSubmitting = false;
		}
	}
</script>

<Card class="mb-6 overflow-hidden border-2 transition-all hover:border-primary/20">
	<CardHeader class="bg-muted/30">
		<CardTitle class="text-xl font-bold">{name}</CardTitle>
		<CardDescription>
			<span
				class={cn(
					"inline-flex items-center rounded-full px-2 py-1 text-xs font-medium ring-1 ring-inset",
					enabled
						? "bg-primary/15 text-primary ring-primary/25"
						: "bg-muted text-muted-foreground ring-border",
				)}
			>
				{enabled ? "Enabled" : "Disabled"}
			</span>
		</CardDescription>
	</CardHeader>
	<CardContent class="pt-6">
		{#if optionsSchema}
			<JsonForm
				schema={optionsSchema}
				initialValue={options}
				onSubmit={handleSubmit}
				isDisabled={isSubmitting}
			/>
		{/if}
	</CardContent>
</Card>

<script lang="ts">
	import Button from "$lib/components/ui/button/button.svelte";
	import Card from "$lib/components/ui/card/card.svelte";
	import CardContent from "$lib/components/ui/card/card-content.svelte";
	import CardDescription from "$lib/components/ui/card/card-description.svelte";
	import CardHeader from "$lib/components/ui/card/card-header.svelte";
	import CardTitle from "$lib/components/ui/card/card-title.svelte";
	import JsonForm from "$lib/JsonForm/JsonForm.svelte";
	import { fetchAdminJson } from "$lib/api";
	import { cn } from "$lib/utils";

	let {
		id,
		userID,
		name,
		enabled,
		options = $bindable(),
		optionsSchema,
	}: {
		id: string;
		userID: string;
		name: string;
		enabled: boolean;
		options: Record<string, any>;
		optionsSchema: Record<string, any>;
	} = $props();

	let isSubmitting = $state(false);

	async function handleSubmit({ value }: { value: any }) {
		isSubmitting = true;
		try {
			// TODO
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
						? "bg-green-100 text-green-700 ring-green-600/20"
						: "bg-gray-100 text-gray-600 ring-gray-500/10",
				)}
			>
				{enabled ? "Enabled" : "Disabled"}
			</span>
		</CardDescription>
	</CardHeader>
	<CardContent class="pt-6">
		<JsonForm schema={optionsSchema} bind:value={options} onSubmit={handleSubmit}>
			<div class="mt-8 flex justify-end">
				<Button type="submit" disabled={isSubmitting} class="min-w-[120px]">
					{#if isSubmitting}
						<span
							class="mr-2 h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"
						></span>
						Saving...
					{:else}
						Save & Enable
					{/if}
				</Button>
			</div>
		</JsonForm>
	</CardContent>
</Card>

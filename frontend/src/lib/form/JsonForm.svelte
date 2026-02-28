<script lang="ts">
	import { BasicForm, createForm, type Schema, type UiSchemaRoot } from "@sjsf/form";
	import "@sjsf/shadcn4-theme/styles.css";
	import { setShadcnContext } from "./theme";

	import { idBuilder, merger, resolver, theme, translation, validator } from "$lib/form/defaults";

	interface Props {
		schema: Schema;
		uiSchema?: UiSchemaRoot;
		initialValue?: unknown;
		onSubmit?: (value: unknown, e: SubmitEvent) => void;
	}
	let { schema, uiSchema, initialValue = {}, onSubmit }: Props = $props();

	setShadcnContext();

	function handleSubmit(value: unknown, e: SubmitEvent) {
		console.log("JsonForm.handleSubmit called", { value, schema, initialValue });
		try {
			onSubmit?.(value, e);
		} catch (error) {
			console.error("Error in onSubmit:", error);
			throw error;
		}
	}

	let form = $derived.by(() => {
		return createForm({
			schema,
			uiSchema,
			initialValue,
			resolver,
			theme,
			idBuilder,
			merger,
			translation,
			validator,
			onSubmit: handleSubmit,
		});
	});
</script>

{#if form}
	{#if form}
		<BasicForm {form} />
	{/if}
{/if}

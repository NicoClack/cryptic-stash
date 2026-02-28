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
		onSubmit?.(value, e);
	}

	let form = $derived.by(() => {
		let correctedSchema = { ...schema };
		// Hack: validation always seems to fail for the first submit
		// if this is specified, at least if it's draft-07
		delete correctedSchema.$schema;

		return createForm({
			schema: correctedSchema,
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
	<BasicForm {form} />
{/if}

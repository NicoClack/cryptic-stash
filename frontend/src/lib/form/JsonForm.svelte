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
		isDisabled?: boolean;
	}
	let { schema, uiSchema, initialValue = {}, onSubmit, isDisabled = false }: Props = $props();

	setShadcnContext();

	function handleSubmit(value: unknown, e: SubmitEvent) {
		onSubmit?.(value, e);
	}

	let correctedSchema = $derived.by(() => {
		let schemaCopy = { ...schema };
		// Hack: validation always seems to fail for the first submit
		// if this is specified, at least if it's draft-07
		delete schemaCopy.$schema;
		return schemaCopy;
	});

	let form = $derived(
		createForm({
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
			disabled: isDisabled,
		}),
	);
</script>

{#if form}
	<div
		class="space-y-4 [&_form]:space-y-4 [&_p]:text-sm [&_p]:text-muted-foreground [&_p]:md:text-base"
	>
		<BasicForm {form} />
	</div>
{/if}

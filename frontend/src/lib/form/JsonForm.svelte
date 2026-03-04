<script lang="ts">
	import { BasicForm, createForm, type Schema, type UiSchemaRoot } from "@sjsf/form";
	import { overrideByRecord } from "@sjsf/form/lib/resolver";
	import "@sjsf/shadcn4-theme/styles.css";

	import { idBuilder, merger, resolver, theme, translation, validator } from "$lib/form/defaults";
	import { setShadcnContext } from "./theme";

	interface Props {
		schema: Schema;
		uiSchema?: UiSchemaRoot;
		initialValue?: unknown;
		onSubmit?: (value: unknown, e: SubmitEvent) => void;
		isDisabled?: boolean;
		submitLabel?: string;
	}
	let {
		schema,
		uiSchema,
		initialValue = {},
		onSubmit,
		isDisabled = false,
		submitLabel,
	}: Props = $props();

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

	let formTranslation = $derived.by(() => {
		if (!submitLabel) {
			return translation;
		}

		return overrideByRecord(translation, {
			submit: submitLabel,
		});
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
			translation: formTranslation,
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

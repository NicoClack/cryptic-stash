<script lang="ts">
	import { createValidator } from "@sjsf/ajv8-validator";
	import { Form, type FormProps } from "@sjsf/form";
	import "@sjsf/shadcn4-theme/styles.css";
	import type { Snippet } from "svelte";
	import { setShadcnContext } from "./theme";

	interface Props extends Omit<FormProps, "validator"> {
		value?: Record<string, any>;
		children?: Snippet;
	}
	let { schema, value = $bindable({}), onSubmit, children, ...rest }: Props = $props();

	setShadcnContext();
	const validator = createValidator();
</script>

<Form {schema} bind:value {onSubmit} {validator} {...rest}>
	{@render children?.()}
</Form>

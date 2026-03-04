import { setThemeContext } from "@sjsf/shadcn4-theme";

import Button from "$lib/components/ui/button/button.svelte";
import Checkbox from "$lib/components/ui/checkbox/checkbox.svelte";
import Input from "$lib/components/ui/input/input.svelte";
import SelectContent from "$lib/components/ui/select/select-content.svelte";
import SelectItem from "$lib/components/ui/select/select-item.svelte";
import SelectTrigger from "$lib/components/ui/select/select-trigger.svelte";
import Select from "$lib/components/ui/select/select.svelte";
import ButtonGroup from "$lib/form/components/ButtonGroup.svelte";
import Field from "$lib/form/components/Field.svelte";
import FieldDescription from "$lib/form/components/FieldDescription.svelte";
import FieldError from "$lib/form/components/FieldError.svelte";
import FieldGroup from "$lib/form/components/FieldGroup.svelte";
import FieldLabel from "$lib/form/components/FieldLabel.svelte";
import FieldLegend from "$lib/form/components/FieldLegend.svelte";
import FieldSet from "$lib/form/components/FieldSet.svelte";
import FieldTitle from "$lib/form/components/FieldTitle.svelte";

export function setShadcnContext() {
	setThemeContext({
		components: {
			ButtonGroup,
			Field,
			FieldLabel,
			FieldError,
			FieldDescription,
			FieldGroup,
			FieldLegend,
			FieldTitle,
			FieldSet,
			Button,
			Checkbox,
			Input,
			Select,
			SelectContent,
			SelectItem,
			SelectTrigger,
		},
	});
}

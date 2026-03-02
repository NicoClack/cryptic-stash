import { setThemeContext } from "@sjsf/shadcn4-theme";

import Button from "$lib/components/ui/button/button.svelte";
import Checkbox from "$lib/components/ui/checkbox/checkbox.svelte";
import Input from "$lib/components/ui/input/input.svelte";
import SelectContent from "$lib/components/ui/select/select-content.svelte";
import SelectItem from "$lib/components/ui/select/select-item.svelte";
import SelectTrigger from "$lib/components/ui/select/select-trigger.svelte";
import Select from "$lib/components/ui/select/select.svelte";
// import { Textarea } from "$lib/components/ui/textarea/index.js";
import RadioGroupItem from "$lib/components/ui/radio-group/radio-group-item.svelte";
import RadioGroup from "$lib/components/ui/radio-group/radio-group.svelte";
/*
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "$lib/components/ui/command/index.js";
import { Calendar } from "$lib/components/ui/calendar/index.js";
import {
  ToggleGroup,
  ToggleGroupItem,
} from "$lib/components/ui/toggle-group/index.js";
import { Slider } from "$lib/components/ui/slider/index.js";
*/
import Switch from "$lib/components/ui/switch/switch.svelte";
/*
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "$lib/components/ui/popover/index.js";
import {
  Field,
  FieldLabel,
  FieldError,
  FieldDescription,
  FieldGroup,
  FieldLegend,
  FieldTitle,
  FieldSet,
} from "$lib/components/ui/field/index.js";
import { ButtonGroup } from "$lib/components/ui/button-group/index.js";
import { RangeCalendar } from "$lib/components/ui/range-calendar/index.js";
*/

export function setShadcnContext() {
	setThemeContext({
		components: {
			// ButtonGroup,
			// Field,
			// FieldLabel,
			// FieldError,
			// FieldDescription,
			// FieldGroup,
			// FieldLegend,
			// FieldTitle,
			// FieldSet,
			Button,
			Checkbox,
			Input,
			Select,
			SelectContent,
			SelectItem,
			SelectTrigger,
			// Textarea,
			RadioGroup,
			RadioGroupItem,
			// Command,
			// CommandEmpty,
			// CommandGroup,
			// CommandInput,
			// CommandItem,
			// CommandList,
			// Calendar,
			// ToggleGroup,
			// ToggleGroupItem,
			// Slider,
			Switch,
			// Popover,
			// PopoverContent,
			// PopoverTrigger,
			// RangeCalendar,
		},
	});
}

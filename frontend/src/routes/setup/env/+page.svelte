<script lang="ts">
	import type { AdminEnvVars } from "$lib/admin/setup";
	import {
		Card,
		CardContent,
		CardDescription,
		CardHeader,
		CardTitle,
	} from "$lib/components/ui/card";
	import Step1 from "./Step1.svelte";
	import Step2 from "./Step2.svelte";
	import Step3 from "./Step3.svelte";
	import Step4 from "./Step4.svelte";

	let step = $state(1);
	let adminEnvVars = $state<AdminEnvVars | null>(null);

	function handleStep1Complete(vars: AdminEnvVars) {
		adminEnvVars = vars;
		step = 2;
	}
	function handleStep2Complete() {
		step = 3;
	}
	function handleStep3Complete(headerName: string) {
		adminEnvVars!.envVars.PROXY_ORIGINAL_IP_HEADER_NAME = headerName;
		adminEnvVars!.envVars.ENABLE_ENV_SETUP = "false";
		step = 4;
	}
</script>

<main>
	<Card>
		<CardHeader>
			<CardTitle class="text-3xl">Welcome to Cryptic Stash</CardTitle>
			<CardDescription class="text-base">
				Before you can set up users and stashes, you need to configure some security-related
				environment variables.
			</CardDescription>
		</CardHeader>
		<CardContent>
			{#if step === 1 || !adminEnvVars}
				<Step1 onComplete={handleStep1Complete}></Step1>
			{:else if step === 2}
				<Step2
					totpSecret={adminEnvVars.envVars.ADMIN_TOTP_SECRET}
					totpURL={adminEnvVars.totpUrl}
					onComplete={handleStep2Complete}
				></Step2>
			{:else if step === 3}
				<Step3 onComplete={handleStep3Complete}></Step3>
			{:else if step === 4}
				<Step4 {adminEnvVars}></Step4>
			{/if}
		</CardContent>
	</Card>
</main>

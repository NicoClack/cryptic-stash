<script lang="ts">
	import { resolve } from "$app/paths";
	import { Card, CardDescription, CardHeader, CardTitle } from "$lib/components/ui/card";

	const homepageLinks = [
		{
			title: "Download your stash",
			description:
				"You'll need your password and somewhere to store the authorisation code while you wait",
			href: resolve("/download-stash"),
		},
		{
			title: "Freeze your account",
			description: "Temporarily block downloads while you're away",
			href: resolve("/freeze-account"),
		},
		{
			title: "Admin site",
			description: "Create, update and lock users",
			href: resolve("/admin/"),
		},
	];
</script>

<main class="mx-auto w-full max-w-3xl space-y-6 px-6 py-10">
	<Card>
		<CardHeader>
			<CardTitle class="text-3xl">
				<h1>Welcome to Cryptic Stash</h1>
			</CardTitle>
			<CardDescription class="text-base">
				Securely store and recover your 2FA recovery codes in case you lose access to your devices.
				If an attacker tries to log in, you'll be notified and can block it.
			</CardDescription>
		</CardHeader>
	</Card>

	<section class="mx-auto flex w-full max-w-5xl flex-col items-center gap-4">
		<a
			href={resolve("/freeze-account")}
			class="group block w-full no-underline outline-none hover:no-underline focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
		>
			<Card>
				<CardHeader>
					<CardTitle
						class="text-center transition-colors group-hover:text-primary group-hover:underline"
					>
						I'm being hacked!!!
					</CardTitle>
				</CardHeader>
			</Card>
		</a>

		<div class="grid w-full gap-4 md:grid-cols-3">
			{#each homepageLinks as item (item.title)}
				<a
					href={item.href}
					class="group block no-underline outline-none hover:no-underline focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
				>
					<Card class="h-full">
						<CardHeader>
							<CardTitle
								class="text-center transition-colors group-hover:text-primary group-hover:underline"
							>
								{item.title}
							</CardTitle>
							<CardDescription>{item.description}</CardDescription>
						</CardHeader>
					</Card>
				</a>
			{/each}
		</div>
	</section>
</main>

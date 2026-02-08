import { maybeGoToSetup } from "$lib/api";
import type { PageLoad } from "./$types";

export const load: PageLoad = async ({ fetch }) => {
	await maybeGoToSetup(fetch);
};

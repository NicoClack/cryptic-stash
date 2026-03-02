import { browser } from "$app/environment";
import { goto } from "$app/navigation";
import { resolve } from "$app/paths";
import { page } from "$app/state";
import { SvelteURL } from "svelte/reactivity";

const ADMIN_SESSION_TOKEN_STORAGE_KEY = "adminSessionToken";
const ADMIN_USER_ID_STORAGE_KEY = "adminUserID";

class AdminAuthState {
	#sessionToken: string | null = $state(null);
	#userID: string | null = $state(null);

	constructor() {
		if (browser) {
			const sessionToken = localStorage.getItem(ADMIN_SESSION_TOKEN_STORAGE_KEY);
			const userID = localStorage.getItem(ADMIN_USER_ID_STORAGE_KEY);
			if (sessionToken && userID) {
				this.#sessionToken = sessionToken;
				this.#userID = userID;
			}
		}
	}

	get userID() {
		return this.#userID;
	}

	isAuthenticated() {
		return this.#sessionToken !== null && this.#userID !== null;
	}
	requireAuth() {
		if (browser && !this.isAuthenticated() && !page.route.id?.startsWith("/admin/login")) {
			const urlObj = new SvelteURL(resolve("/admin/login"), location.origin);
			urlObj.searchParams.set("redirectTo", page.url.pathname + page.url.search);
			// eslint-disable-next-line svelte/no-navigation-without-resolve
			goto(urlObj.toString());
		}
	}
	getAuthHeader(): string | null {
		if (!this.isAuthenticated()) {
			return null;
		}

		return `AdminCode ${this.#sessionToken}`;
	}
	login(sessionToken: string, userID: string) {
		this.#sessionToken = sessionToken;
		this.#userID = userID;
		localStorage.setItem(ADMIN_SESSION_TOKEN_STORAGE_KEY, sessionToken);
		localStorage.setItem(ADMIN_USER_ID_STORAGE_KEY, userID);

		const redirectTo = page.url.searchParams.get("redirectTo");
		if (redirectTo) {
			const urlObj = new SvelteURL(redirectTo, location.origin);
			if (urlObj.origin === location.origin) {
				// eslint-disable-next-line svelte/no-navigation-without-resolve
				goto(urlObj.toString());
				return;
			}
		}
		goto(resolve("/admin"));
	}
}

export const adminAuth = new AdminAuthState();

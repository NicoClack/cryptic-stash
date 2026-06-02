import { browser } from "$app/environment";
import { goto } from "$app/navigation";
import { resolve } from "$app/paths";
import { page } from "$app/state";
import { goToLogin } from "$lib/api";
import { SvelteURL } from "svelte/reactivity";

const USER_SESSION_TOKEN_STORAGE_KEY = "userSessionToken";
const USER_USER_ID_STORAGE_KEY = "userUserID";

class UserAuthState {
	#sessionToken: string | null = $state(null);
	#userID: string | null = $state(null);

	constructor() {
		if (browser) {
			const sessionToken = sessionStorage.getItem(USER_SESSION_TOKEN_STORAGE_KEY);
			const userID = sessionStorage.getItem(USER_USER_ID_STORAGE_KEY);
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
		if (browser && !this.isAuthenticated() && !page.route.id?.startsWith("/login")) {
			goToLogin();
		}
	}
	getAuthHeader(): string | null {
		if (!this.isAuthenticated()) {
			return null;
		}

		return `Session ${this.#sessionToken}`;
	}
	login(sessionToken: string, userID: string) {
		this.#sessionToken = sessionToken;
		this.#userID = userID;
		sessionStorage.setItem(USER_SESSION_TOKEN_STORAGE_KEY, sessionToken);
		sessionStorage.setItem(USER_USER_ID_STORAGE_KEY, userID);

		const redirectTo = page.url.searchParams.get("redirectTo");
		if (redirectTo) {
			const urlObj = new SvelteURL(redirectTo, location.origin);
			if (urlObj.origin === location.origin) {
				goto(urlObj.toString());
				return;
			}
		}
		goto(resolve("/"));
	}
}

export const userAuth = new UserAuthState();

import { browser } from "$app/environment";
import { goto } from "$app/navigation";
import { resolve } from "$app/paths";
import { page } from "$app/state";
import { goToElevate, goToLogin } from "$lib/api";
import { SvelteURL } from "svelte/reactivity";

// TODO: rework to store as a single object in sessionStorage instead of multiple keys
const USER_SESSION_TOKEN_STORAGE_KEY = "userSessionToken";
const USER_USER_ID_STORAGE_KEY = "userUserID";
const USER_USERNAME_STORAGE_KEY = "userUsername";
const USER_SUPER_USER_MODE_STORAGE_KEY = "userSuperUserMode";

class UserAuthState {
	#sessionToken: string | null = $state(null);
	#userID: string | null = $state(null);
	#username: string | null = $state(null);
	#isSuperUserMode: boolean = $state(false);

	constructor() {
		if (browser) {
			const sessionToken = sessionStorage.getItem(USER_SESSION_TOKEN_STORAGE_KEY);
			const userID = sessionStorage.getItem(USER_USER_ID_STORAGE_KEY);
			const username = sessionStorage.getItem(USER_USERNAME_STORAGE_KEY);
			const isSuperUserMode = sessionStorage.getItem(USER_SUPER_USER_MODE_STORAGE_KEY);
			if (sessionToken && userID) {
				this.#sessionToken = sessionToken;
				this.#userID = userID;
				this.#username = username;
				this.#isSuperUserMode = isSuperUserMode === "true";
			}
		}
	}

	get userID() {
		return this.#userID;
	}
	get username() {
		return this.#username;
	}
	get superUserMode() {
		return this.#isSuperUserMode;
	}

	isAuthenticated() {
		return this.#sessionToken !== null && this.#userID !== null;
	}
	isSuperUserMode() {
		return this.isAuthenticated() && this.#isSuperUserMode;
	}

	requireAuth(): boolean {
		if (browser && !this.isAuthenticated() && !page.route.id?.startsWith("/login")) {
			goToLogin();
			return false;
		}
		return true;
	}
	requireSuperUserMode() {
		if (
			browser &&
			this.requireAuth() &&
			!this.#isSuperUserMode &&
			!page.route.id?.startsWith("/elevate")
		) {
			goToElevate();
			return false;
		}
		return true;
	}

	getAuthHeader(): string | null {
		if (!this.isAuthenticated()) {
			return null;
		}

		return `Bearer ${this.#sessionToken}`;
	}

	login(sessionToken: string, userID: string, username: string, isSuperUserMode: boolean) {
		this.#sessionToken = sessionToken;
		this.#userID = userID;
		this.#username = username;
		this.#isSuperUserMode = isSuperUserMode;
		sessionStorage.setItem(USER_SESSION_TOKEN_STORAGE_KEY, sessionToken);
		sessionStorage.setItem(USER_USER_ID_STORAGE_KEY, userID);
		sessionStorage.setItem(USER_USERNAME_STORAGE_KEY, username);
		sessionStorage.setItem(USER_SUPER_USER_MODE_STORAGE_KEY, this.#isSuperUserMode.toString());

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
	elevate() {
		this.#isSuperUserMode = true;
		sessionStorage.setItem(USER_SUPER_USER_MODE_STORAGE_KEY, this.#isSuperUserMode.toString());
	}
	logout() {
		this.#sessionToken = null;
		this.#userID = null;
		this.#username = null;
		this.#isSuperUserMode = false;
		sessionStorage.removeItem(USER_SESSION_TOKEN_STORAGE_KEY);
		sessionStorage.removeItem(USER_USER_ID_STORAGE_KEY);
		sessionStorage.removeItem(USER_USERNAME_STORAGE_KEY);
		sessionStorage.removeItem(USER_SUPER_USER_MODE_STORAGE_KEY);
	}
}

export const userAuth = new UserAuthState();

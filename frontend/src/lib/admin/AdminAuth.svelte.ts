import { browser } from "$app/environment";
import { goto } from "$app/navigation";
import { resolve } from "$app/paths";
import { page } from "$app/state";
import { createContext } from "svelte";

export class AdminAuthState {
	#sessionToken: string | null = $state(null);
	#userID: string | null = $state(null);

	constructor() {
		if (browser) {
			const sessionToken = localStorage.getItem("adminSessionToken");
			const userID = localStorage.getItem("adminUserID");
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
			goto(resolve("/admin/login"));
		}
	}
	getAuthHeader(): string | null {
		if (!this.isAuthenticated()) {
			return null;
		}

		return `AdminCode ${this.#sessionToken}`;
	}
}

export const [getAdminAuth, setAdminAuth] = createContext<AdminAuthState>();

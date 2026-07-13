import { goto } from "$app/navigation";
import { resolve } from "$app/paths";
import { page } from "$app/state";
import { PUBLIC_API_DOMAIN } from "$env/static/public";
import { SvelteURL } from "svelte/reactivity";
import { adminAuth } from "./admin/AdminAuth.svelte";
import { userAuth } from "./auth/UserAuth.svelte";

class StatusError extends Error {
	jsonResponse: JsonResponse;

	constructor(resp: JsonResponse) {
		super(`request failed with status ${resp.status}`);
		this.jsonResponse = resp;
	}
}

export class JsonResponse {
	headers: Headers;
	status: number;
	data: any;
	redirecting: boolean;

	constructor(resp: Response, data: any) {
		this.headers = resp.headers;
		this.status = resp.status;
		this.data = data;
		this.redirecting = false;
	}

	get ok(): boolean {
		return this.status >= 200 && this.status <= 299;
	}
	throwForStatus() {
		if (!this.ok) {
			throw new StatusError(this);
		}
	}
}
export interface InnerResponse {
	errors?: ApiErrorDetail[];
}
export interface ApiErrorDetail {
	code: string;
	message: string;
}

export interface JsonResponseInit extends RequestInit {
	throwForStatus?: boolean;
}
export async function fetchJson(
	fetch: typeof global.fetch,
	url: string,
	init?: JsonResponseInit | undefined,
): Promise<JsonResponse> {
	const urlObj = new URL(PUBLIC_API_DOMAIN + url, window.location.origin);
	const resp = await fetch(urlObj, init);
	const json = await resp.json();

	const jsonResponse = new JsonResponse(resp, json);
	if (
		resp.status === 404 &&
		responseHasErrorCode(jsonResponse, "ENDPOINT_NOT_FOUND") &&
		!page.route.id?.startsWith("/setup") &&
		!urlObj.pathname.startsWith("/api/v1/setup/")
	) {
		if (await maybeGoToSetup(fetch)) {
			jsonResponse.redirecting = true;
		}
	}
	if (
		(resp.status === 401 || resp.status === 403) &&
		page.route.id !== "/admin/login" &&
		page.route.id !== "/login"
	) {
		const authHeader = new Headers(init?.headers).get("authorization");

		if (resp.status === 403 && responseHasErrorCode(jsonResponse, "SUDO_MODE_REQUIRED")) {
			jsonResponse.redirecting = true;
			goToElevate();
		}

		if (page.route.id?.startsWith("/admin") || page.route.id === "/setup/admin-messengers") {
			jsonResponse.redirecting = true;
			goToAdminLogin();
		} else if (authHeader?.startsWith("Bearer ")) {
			// TODO: ^ how do I distinguish between user and admin auth if they both use Bearer tokens?
			jsonResponse.redirecting = true;
			userAuth.logout();
			goToLogin();
		}
	}
	if (init?.throwForStatus) {
		jsonResponse.throwForStatus();
	}

	return jsonResponse;
}
export async function fetchAdminJson(
	fetch: typeof global.fetch,
	url: string,
	init?: JsonResponseInit | undefined,
): Promise<JsonResponse> {
	adminAuth.requireAuth();

	const headers = new Headers(init?.headers);
	const authHeader = adminAuth.getAuthHeader();
	if (authHeader) {
		headers.set("Authorization", authHeader);
	}

	return await fetchJson(fetch, url, {
		...init,
		headers: headers,
	});
}

export async function fetchUserJson(
	fetch: typeof global.fetch,
	url: string,
	init?: JsonResponseInit | undefined,
): Promise<JsonResponse> {
	userAuth.requireAuth();

	const headers = new Headers(init?.headers);
	const authHeader = userAuth.getAuthHeader();
	if (authHeader) {
		headers.set("Authorization", authHeader);
	}

	return await fetchJson(fetch, url, {
		...init,
		headers: headers,
	});
}

export function responseHasErrorCode(response: JsonResponse, errorCode: string): boolean {
	const errors = response.data?.errors;
	if (!Array.isArray(errors)) return false;

	return errors.find((error) => error?.code === errorCode) != null;
}

export async function maybeGoToSetup(fetch: typeof global.fetch): Promise<boolean> {
	const setupStatus = await fetchJson(fetch, "/api/v1/setup/");
	setupStatus.throwForStatus();
	if (setupStatus.data.isComplete) {
		return false;
	}
	if (!setupStatus.data.isEnvComplete) {
		goto(resolve("/setup/env/"));
	} else {
		goto(resolve("/setup/admin-messengers/"));
	}
	return true;
}
export function goToAdminLogin(): void {
	const urlObj = new SvelteURL(resolve("/admin/login"), location.origin);
	urlObj.searchParams.set("redirectTo", page.url.pathname + page.url.search);
	goto(urlObj.toString());
}
export function goToLogin(): void {
	const urlObj = new SvelteURL(resolve("/login"), location.origin);
	urlObj.searchParams.set("redirectTo", page.url.pathname + page.url.search);
	goto(urlObj.toString());
}
export function goToElevate(): void {
	const urlObj = new SvelteURL(resolve("/elevate"), location.origin);
	urlObj.searchParams.set("redirectTo", page.url.pathname + page.url.search);
	goto(urlObj.toString());
}

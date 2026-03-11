import { goto } from "$app/navigation";
import { resolve } from "$app/paths";
import { page } from "$app/state";
import { PUBLIC_API_DOMAIN } from "$env/static/public";
import { adminAuth } from "./admin/AdminAuth.svelte";

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
		((page.route.id?.startsWith("/admin") && page.route.id !== "/admin/login") ||
			page.route.id === "/setup/admin-messengers")
	) {
		goto(resolve("/admin/login"));
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

import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}
export type WithElementRef<T> = T & {
	ref?: any;
};
export type { WithoutChild, WithoutChildrenOrChild } from "bits-ui";

export function formatTime(dateValue: string): string {
	const date = new Date(dateValue);
	return date.toLocaleString();
}
export function normalizeEmail(value: string): string {
	return value.trim().toLowerCase();
}

export function encodeBase64UrlFormat(buffer: ArrayBuffer): string {
	const bytes = new Uint8Array(buffer);
	let binary = "";
	for (const byte of bytes) {
		binary += String.fromCharCode(byte);
	}
	return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=/g, "");
}
export function decodeBase64UrlFormat(str: string): ArrayBuffer {
	const padded = str + "=".repeat((4 - (str.length % 4)) % 4);
	const binary = atob(padded.replace(/-/g, "+").replace(/_/g, "/"));
	const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));
	return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength);
}

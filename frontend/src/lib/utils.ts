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

import { writable } from 'svelte/store';

export type BreadcrumbItem = {
	id: string;
	name: string;
};

export const breadcrumbItems = writable<BreadcrumbItem[]>([]);

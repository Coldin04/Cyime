import { apiFetch } from '$lib/api';

export type UserProfile = {
	id: string;
	displayName: string | null;
	email: string | null;
	avatarUrl: string | null;
};

export type UserOverview = {
	activeDocumentCount: number;
	trashedDocumentCount: number;
	documentLimit: number | null;
	unlimited: boolean;
};

async function parseUserResponse(response: Response): Promise<UserProfile> {
	if (!response.ok) {
		const error = await response.json().catch(() => ({}));
		throw new Error(error.error || error.message || 'Failed to update user profile');
	}

	return response.json();
}

export async function updateDisplayName(displayName: string): Promise<UserProfile> {
	const response = await apiFetch('/api/v1/user/profile', {
		method: 'PUT',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({ displayName })
	});

	return parseUserResponse(response);
}

export async function uploadAvatar(file: File): Promise<UserProfile> {
	const formData = new FormData();
	formData.set('file', file);

	const response = await apiFetch('/api/v1/user/avatar', {
		method: 'POST',
		body: formData
	});

	return parseUserResponse(response);
}

export async function setGitHubAvatar(username: string): Promise<UserProfile> {
	const response = await apiFetch('/api/v1/user/avatar/github', {
		method: 'PUT',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({ username })
	});

	return parseUserResponse(response);
}

export async function getUserOverview(): Promise<UserOverview> {
	const response = await apiFetch('/api/v1/user/overview');

	if (!response.ok) {
		const error = await response.json().catch(() => ({}));
		throw new Error(error.error || error.message || 'Failed to load user overview');
	}

	return response.json();
}

import { apiFetch } from '$lib/api';

export type DocumentContent = {
	id: string;
	documentId: string;
	content: string;
	contentJson: string;
	contentMarkdown: string;
	plainText: string;
	createdAt: string;
	updatedAt: string;
};

export type UpdateContentResponse = {
	success: boolean;
	updatedAt: string;
};

export async function getDocumentContent(documentId: string): Promise<DocumentContent> {
	const response = await apiFetch(`/api/v1/edit/documents/${documentId}/content`);

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to fetch document content');
	}

	return response.json();
}

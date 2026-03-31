import { Server } from '@hocuspocus/server';
import * as Y from 'yjs';
import jwt from 'jsonwebtoken';
import axios from 'axios';
import dotenv from 'dotenv';

dotenv.config();

// 配置
const PORT = parseInt(process.env.PORT || '3001', 10);
const GO_API_URL = process.env.GO_API_URL || 'http://localhost:8080/api/v1';
const JWT_SECRET = process.env.JWT_SECRET || 'your-secret-key';

interface TokenPayload {
	sub: string;
	email?: string;
	[key: string]: unknown;
}

interface UserACL {
	myRole: 'viewer' | 'editor' | 'collaborator' | 'owner';
	canRead: boolean;
	canEdit: boolean;
	canManageMembers: boolean;
}

interface RealtimeContext {
	userId: string;
	token: string;
	documentId: string;
	acl: UserACL;
}

function verifyJWT(token: string): TokenPayload | null {
	try {
		return jwt.verify(token, JWT_SECRET) as TokenPayload;
	} catch (error) {
		console.error('JWT verification failed:', error);
		return null;
	}
}

function extractTokenFromUrl(url?: string): string | null {
	if (!url) return null;
	try {
		const parsed = new URL(`ws://localhost${url}`);
		return parsed.searchParams.get('token');
	} catch (error) {
		console.error('Failed to parse token from URL:', error);
		return null;
	}
}

async function getUserACL(documentId: string, token: string): Promise<UserACL | null> {
	try {
		const response = await axios.get(`${GO_API_URL}/workspace/documents/${documentId}/acl`, {
			headers: {
				Authorization: `Bearer ${token}`
			},
			timeout: 5000
		});
		return response.data as UserACL;
	} catch (error) {
		console.error(`Failed to get user ACL for doc ${documentId}:`, error);
		return null;
	}
}

async function loadYjsState(
	documentId: string,
	token: string
): Promise<{ yjsState: string; yjsStateVector: string }> {
	try {
		const response = await axios.get(`${GO_API_URL}/realtime/documents/${documentId}/state`, {
			headers: {
				Authorization: `Bearer ${token}`
			},
			timeout: 5000
		});
		return response.data;
	} catch (error) {
		console.error(`Failed to load Yjs state for doc ${documentId}:`, error);
		return { yjsState: '', yjsStateVector: '' };
	}
}

async function saveYjsState(
	documentId: string,
	token: string,
	yjsState: string,
	yjsStateVector: string
): Promise<void> {
	try {
		await axios.put(
			`${GO_API_URL}/realtime/documents/${documentId}/state`,
			{
				yjsState,
				yjsStateVector
			},
			{
				headers: {
					Authorization: `Bearer ${token}`
				},
				timeout: 5000
			}
		);
	} catch (error) {
		console.error(`Failed to save Yjs state for doc ${documentId}:`, error);
	}
}

function normalizeDocumentId(rawName?: string): string {
	if (!rawName) return '';
	return rawName.startsWith('doc:') ? rawName.slice(4) : rawName;
}

const server = new Server({
	port: PORT,
	address: '0.0.0.0',
	timeout: 30000,

	// 认证 - 从 WebSocket URL 或 token 字段提取 JWT
	async onAuthenticate(data: any) {
		const token = (data?.token as string | undefined) || extractTokenFromUrl(data?.request?.url);

		if (!token) {
			throw new Error('No authentication token provided');
		}

		const payload = verifyJWT(token);
		if (!payload?.sub) {
			throw new Error('Invalid or expired token');
		}

		const documentId = normalizeDocumentId(data?.documentName);
		if (!documentId) {
			throw new Error('Missing document ID');
		}

		const acl = await getUserACL(documentId, token);
		if (!acl?.canRead) {
			const error = new Error('No read permission for this document') as Error & {
				reason?: string;
			};
			error.reason = 'permission-denied';
			throw error;
		}

		data.connectionConfig.readOnly = !acl.canEdit;

		return {
			userId: payload.sub,
			token,
			documentId,
			acl
		};
	},

	// 连接建立前，认证上下文尚未写入；这里只保留轻量日志。
	async onConnect(data: any) {
		const documentId = normalizeDocumentId(data?.documentName);
		console.log(`[DOC:${documentId}] Incoming connection ${data?.socketId ?? 'unknown-socket'}`);
	},

	// 加载文档 - 从 Go API 拉 Yjs state
	async onLoadDocument(data: any) {
		const document = data.document as Y.Doc;
		const context = data.context as RealtimeContext | undefined;
		const documentId = context?.documentId || normalizeDocumentId(data?.documentName);
		const token = context?.token;

		if (!documentId || !token) {
			console.warn('Missing context for loading document');
			return;
		}

		const state = await loadYjsState(documentId, token);
		if (state.yjsState) {
			Y.applyUpdate(document, Buffer.from(state.yjsState, 'base64'));
			console.log(`[DOC:${documentId}] Loaded Yjs state from database`);
		} else {
			console.log(`[DOC:${documentId}] No existing Yjs state found, starting fresh`);
		}
	},

	// 保存文档 - 节流保存到 Go API
	async onStoreDocument(data: any) {
		const document = data.document as Y.Doc;
		const context = data.context as RealtimeContext | undefined;
		const documentId = context?.documentId || normalizeDocumentId(data?.documentName);
		const userId = context?.userId;
		const token = context?.token;
		const acl = context?.acl;

		if (!acl?.canEdit) {
			console.warn(
				`[DOC:${documentId}] User ${userId ?? 'unknown'} attempted write without edit permission`
			);
			return;
		}

		if (!documentId || !token) {
			console.warn('Missing context for storing document');
			return;
		}

		const yjsState = Buffer.from(Y.encodeStateAsUpdate(document)).toString('base64');
		const yjsStateVector = Buffer.from(Y.encodeStateVector(document)).toString('base64');

		await saveYjsState(documentId, token, yjsState, yjsStateVector);
	},

	async onDisconnect(data: any) {
		const context = data.context as RealtimeContext | undefined;
		const documentId = context?.documentId || normalizeDocumentId(data?.documentName);
		const userId = context?.userId;

		if (documentId && userId) {
			console.log(`[DOC:${documentId}] User ${userId} disconnected`);
		}
	}
});

server
	.listen()
	.then(() => {
		console.log(`🚀 Realtime server listening on port ${PORT}`);
		console.log(`📡 WebSocket endpoint: ws://0.0.0.0:${PORT}`);
		console.log(`🔗 Backend API: ${GO_API_URL}`);
	})
	.catch((error: unknown) => {
		console.error('Failed to start server:', error);
		process.exit(1);
	});

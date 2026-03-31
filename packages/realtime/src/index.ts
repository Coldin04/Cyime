import { Server } from '@hocuspocus/server';
import * as Y from 'yjs';
import jwt from 'jsonwebtoken';
import axios from 'axios';
import dotenv from 'dotenv';
import type { IncomingMessage, ServerResponse } from 'node:http';
import { WebSocketServer, WebSocket } from 'ws';

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
	socketId?: string;
}

const collaborationSockets = new Map<string, Set<string>>();
const documentPresence = new Map<string, Map<string, number>>();
const presenceSubscribers = new Map<string, Set<WebSocket>>();
const presenceClientMeta = new WeakMap<WebSocket, { token: string; userId: string; documentId?: string }>();
const presenceWebSocketServer = new WebSocketServer({ noServer: true });

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

function setCORSHeaders(request: IncomingMessage, response: ServerResponse) {
	const origin = request.headers.origin;
	response.setHeader('Access-Control-Allow-Origin', origin ?? '*');
	response.setHeader('Vary', 'Origin');
	response.setHeader('Access-Control-Allow-Credentials', 'true');
	response.setHeader('Access-Control-Allow-Headers', 'Authorization, Content-Type');
	response.setHeader('Access-Control-Allow-Methods', 'GET, OPTIONS');
}

function addCollaborationSocket(documentId: string, socketId: string) {
	let members = collaborationSockets.get(documentId);
	if (!members) {
		members = new Set<string>();
		collaborationSockets.set(documentId, members);
	}
	members.add(socketId);
}

function removeCollaborationSocket(documentId: string, socketId: string) {
	const members = collaborationSockets.get(documentId);
	if (!members) {
		return;
	}
	members.delete(socketId);
	if (members.size === 0) {
		collaborationSockets.delete(documentId);
	}
}

function getCollaborationSocketCount(documentId: string): number {
	return collaborationSockets.get(documentId)?.size ?? 0;
}

function addDocumentPresence(documentId: string, userId: string) {
	let users = documentPresence.get(documentId);
	if (!users) {
		users = new Map<string, number>();
		documentPresence.set(documentId, users);
	}

	users.set(userId, (users.get(userId) ?? 0) + 1);
}

function removeDocumentPresence(documentId: string, userId: string) {
	const users = documentPresence.get(documentId);
	if (!users) {
		return;
	}

	const nextCount = (users.get(userId) ?? 0) - 1;
	if (nextCount > 0) {
		users.set(userId, nextCount);
		return;
	}

	users.delete(userId);
	if (users.size === 0) {
		documentPresence.delete(documentId);
	}
}

function getDocumentPresenceCount(documentId: string): number {
	return documentPresence.get(documentId)?.size ?? 0;
}

function broadcastPresence(documentId: string) {
	const subscribers = presenceSubscribers.get(documentId);
	if (!subscribers || subscribers.size === 0) {
		return;
	}

	const payload = JSON.stringify({
		type: 'presence',
		documentId,
		connectedCount: getDocumentPresenceCount(documentId)
	});

	for (const subscriber of subscribers) {
		if (subscriber.readyState === WebSocket.OPEN) {
			subscriber.send(payload);
		}
	}
}

function removePresenceSubscriber(socket: WebSocket) {
	const meta = presenceClientMeta.get(socket);
	if (!meta?.documentId) {
		return;
	}

	removeDocumentPresence(meta.documentId, meta.userId);
	broadcastPresence(meta.documentId);

	const subscribers = presenceSubscribers.get(meta.documentId);
	if (!subscribers) {
		return;
	}

	subscribers.delete(socket);
	if (subscribers.size === 0) {
		presenceSubscribers.delete(meta.documentId);
	}
}

const server = new Server({
	port: PORT,
	address: '0.0.0.0',
	timeout: 30000,
	async onUpgrade(data: any) {
		const request = data.request as IncomingMessage;
		const requestURL = new URL(request.url ?? '/', `http://${request.headers.host ?? 'localhost'}`);
		if (requestURL.pathname !== '/api/v1/realtime/presence/ws') {
			return;
		}

		presenceWebSocketServer.handleUpgrade(data.request, data.socket, data.head, (ws: WebSocket) => {
			presenceWebSocketServer.emit('connection', ws, data.request);
		});

		throw null;
	},

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
			acl,
			socketId: data?.socketId
		};
	},

	// 连接建立前，认证上下文尚未写入；这里只保留轻量日志。
	async onConnect(data: any) {
		const documentId = normalizeDocumentId(data?.documentName);
		console.log(`[DOC:${documentId}] Incoming connection ${data?.socketId ?? 'unknown-socket'}`);
	},

	async connected(data: any) {
		const context = data.context as RealtimeContext | undefined;
		const documentId = context?.documentId || normalizeDocumentId(data?.documentName);
		const socketId = context?.socketId || data?.socketId;

		if (documentId && socketId) {
			addCollaborationSocket(documentId, socketId);
		}

		if (documentId && context?.userId) {
			console.log(
				`[DOC:${documentId}] User ${context.userId} connected with role: ${context.acl.myRole} (${getCollaborationSocketCount(documentId)} collab sockets)`
			);
		}
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
		const socketId = context?.socketId || data?.socketId;

		if (documentId && socketId) {
			removeCollaborationSocket(documentId, socketId);
		}

		if (documentId && userId) {
			console.log(`[DOC:${documentId}] User ${userId} disconnected (${getCollaborationSocketCount(documentId)} collab sockets)`);
		}
	},

	async onRequest(data: any) {
		const request = data.request as IncomingMessage;
		const response = data.response as ServerResponse;
		const requestURL = new URL(request.url ?? '/', `http://${request.headers.host ?? 'localhost'}`);

		setCORSHeaders(request, response);

		if (request.method === 'OPTIONS') {
			response.writeHead(204);
			response.end();
			throw null;
		}

		if (requestURL.pathname !== '/api/v1/realtime/presence') {
			return;
		}

		const documentId = normalizeDocumentId(requestURL.searchParams.get('documentId') ?? '');
		if (!documentId) {
			response.writeHead(400, { 'Content-Type': 'application/json' });
			response.end(JSON.stringify({ error: 'documentId is required' }));
			throw null;
		}

		const authHeader = request.headers.authorization ?? '';
		const token = authHeader.startsWith('Bearer ') ? authHeader.slice(7).trim() : '';
		const payload = token ? verifyJWT(token) : null;
		if (!payload?.sub) {
			response.writeHead(401, { 'Content-Type': 'application/json' });
			response.end(JSON.stringify({ error: 'Unauthorized' }));
			throw null;
		}

		const acl = await getUserACL(documentId, token);
		if (!acl?.canRead) {
			response.writeHead(403, { 'Content-Type': 'application/json' });
			response.end(JSON.stringify({ error: 'Forbidden' }));
			throw null;
		}

		response.writeHead(200, { 'Content-Type': 'application/json' });
		response.end(
			JSON.stringify({
				documentId,
				connectedCount: getDocumentPresenceCount(documentId),
				hasCollaboration: getCollaborationSocketCount(documentId) > 0
			})
		);
		throw null;
	}
});

presenceWebSocketServer.on('connection', (socket: WebSocket, request: IncomingMessage) => {
	const requestURL = new URL(request.url ?? '/', `http://${request.headers.host ?? 'localhost'}`);
	const token = requestURL.searchParams.get('token')?.trim() ?? '';
	const payload = token ? verifyJWT(token) : null;

	if (!payload?.sub) {
		socket.close(4401, 'unauthorized');
		return;
	}

	presenceClientMeta.set(socket, { token, userId: payload.sub });

	socket.on('message', async (raw: Buffer) => {
		try {
			const message = JSON.parse(raw.toString()) as { type?: string; documentId?: string };
			if (message.type !== 'subscribe' || !message.documentId) {
				return;
			}

			const meta = presenceClientMeta.get(socket);
			if (!meta?.token) {
				socket.close(4401, 'unauthorized');
				return;
			}

			const documentId = normalizeDocumentId(message.documentId);
			const acl = await getUserACL(documentId, meta.token);
			if (!acl?.canRead) {
				socket.close(4403, 'forbidden');
				return;
			}

			removePresenceSubscriber(socket);

			let subscribers = presenceSubscribers.get(documentId);
			if (!subscribers) {
				subscribers = new Set<WebSocket>();
				presenceSubscribers.set(documentId, subscribers);
			}
			subscribers.add(socket);
			presenceClientMeta.set(socket, { ...meta, documentId });
			addDocumentPresence(documentId, meta.userId);
			broadcastPresence(documentId);
		} catch (error) {
			console.error('[Presence] Failed to handle message:', error);
		}
	});

	socket.on('close', () => {
		removePresenceSubscriber(socket);
	});
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

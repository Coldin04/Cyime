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
const JWT_SECRET = (() => {
	const secret = process.env.JWT_SECRET_KEY || process.env.JWT_SECRET;
	if (!secret) {
		throw new Error('JWT secret not configured. Please set JWT_SECRET_KEY or JWT_SECRET.');
	}
	return secret;
})();

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
	// yjsVersion is the optimistic-concurrency token for this document; the
	// realtime server tracks the version it last persisted and echoes it on
	// the next save so the Go API can reject stale or racing writes.
	yjsVersion: number;
}

// ACL re-validation cache. We can't trust the ACL captured at connect time —
// the document owner may revoke access mid-session. A 30s TTL keeps the cost
// down (one fetch per document per 30s per user) while still bounding how
// long a revoked user can keep editing.
const ACL_CACHE_TTL_MS = 30_000;
const aclCache = new Map<string, { acl: UserACL; expiresAt: number }>();

function aclCacheKey(documentId: string, userId: string): string {
	return `${userId}:${documentId}`;
}

async function getUserACLFresh(
	documentId: string,
	userId: string,
	token: string
): Promise<UserACL | null> {
	const key = aclCacheKey(documentId, userId);
	const cached = aclCache.get(key);
	const now = Date.now();
	if (cached && cached.expiresAt > now) {
		return cached.acl;
	}
	const acl = await getUserACL(documentId, token);
	if (acl) {
		aclCache.set(key, { acl, expiresAt: now + ACL_CACHE_TTL_MS });
	}
	return acl;
}

function invalidateACLCache(documentId: string, userId: string): void {
	aclCache.delete(aclCacheKey(documentId, userId));
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
): Promise<{ yjsState: string; yjsStateVector: string; yjsVersion: number }> {
	try {
		const response = await axios.get(`${GO_API_URL}/realtime/documents/${documentId}/state`, {
			headers: {
				Authorization: `Bearer ${token}`
			},
			timeout: 5000
		});
		const data = response.data ?? {};
		return {
			yjsState: typeof data.yjsState === 'string' ? data.yjsState : '',
			yjsStateVector: typeof data.yjsStateVector === 'string' ? data.yjsStateVector : '',
			// New rows from a freshly-migrated DB have yjs_version = 1; the
			// "no row exists yet" path is signalled by a 404 (caught below)
			// and we return 0 so the first save can create the row.
			yjsVersion: typeof data.yjsVersion === 'number' ? data.yjsVersion : 0
		};
	} catch (error) {
		// 404 means the row does not exist yet; that's a normal "fresh
		// document" state and the first save should create it. For any other
		// error we still return zeros, but log loudly so it's not silent.
		const status =
			error && typeof error === 'object' && 'response' in error
				? (error as { response?: { status?: number } }).response?.status
				: undefined;
		if (status !== 404) {
			console.error(`Failed to load Yjs state for doc ${documentId}:`, error);
		}
		return { yjsState: '', yjsStateVector: '', yjsVersion: 0 };
	}
}

class YjsSaveConflictError extends Error {
	currentVersion: number;

	constructor(currentVersion: number) {
		super(`yjs version conflict (current ${currentVersion})`);
		this.name = 'YjsSaveConflictError';
		this.currentVersion = currentVersion;
	}
}

async function saveYjsState(
	documentId: string,
	token: string,
	yjsState: string,
	yjsStateVector: string,
	expectedYjsVersion: number
): Promise<number> {
	try {
		const response = await axios.put(
			`${GO_API_URL}/realtime/documents/${documentId}/state`,
			{
				yjsState,
				yjsStateVector,
				expectedYjsVersion
			},
			{
				headers: {
					Authorization: `Bearer ${token}`
				},
				timeout: 5000,
				// Treat 4xx as a value, not a thrown error, so we can branch on
				// 409 without losing the response body.
				validateStatus: (status) => status >= 200 && status < 500
			}
		);

		if (response.status === 409) {
			const current =
				typeof response.data?.currentYjsVersion === 'number'
					? response.data.currentYjsVersion
					: expectedYjsVersion;
			throw new YjsSaveConflictError(current);
		}
		if (response.status >= 400) {
			const body = response.data ? JSON.stringify(response.data) : '';
			throw new Error(`Yjs save failed with status ${response.status}: ${body}`);
		}

		const newVersion =
			typeof response.data?.yjsVersion === 'number'
				? response.data.yjsVersion
				: expectedYjsVersion + 1;
		return newVersion;
	} catch (error) {
		// Re-throw so Hocuspocus marks the document as still-dirty and retries
		// on the next debounce window. Swallowing the error here is what made
		// the previous version silently lose edits.
		if (error instanceof YjsSaveConflictError) {
			throw error;
		}
		if (error instanceof Error) {
			throw error;
		}
		throw new Error(`Yjs save failed: ${String(error)}`);
	}
}

function normalizeDocumentId(rawName?: string): string {
	if (!rawName) return '';
	return rawName.startsWith('doc:') ? rawName.slice(4) : rawName;
}

function setCORSHeaders(request: IncomingMessage, response: ServerResponse) {
	const origin = request.headers.origin;
	if (origin) {
		response.setHeader('Access-Control-Allow-Origin', origin);
		response.setHeader('Access-Control-Allow-Credentials', 'true');
	} else {
		response.setHeader('Access-Control-Allow-Origin', '*');
	}
	response.setHeader('Vary', 'Origin');
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
	debounce: 60000,
	maxDebounce: 60000,
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

		// Force a fresh fetch on connect so the cache cannot replay a stale
		// "canEdit" decision from a previous session.
		invalidateACLCache(documentId, payload.sub);
		const acl = await getUserACLFresh(documentId, payload.sub, token);
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
			socketId: data?.socketId,
			yjsVersion: 0
		} satisfies RealtimeContext;
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
			console.log(
				`[DOC:${documentId}] Loaded Yjs state from database (version ${state.yjsVersion})`
			);
		} else {
			console.log(`[DOC:${documentId}] No existing Yjs state found, starting fresh`);
		}
		// Stash the version on the connection context so the next save can
		// echo it back as the optimistic-concurrency token.
		if (context) {
			context.yjsVersion = state.yjsVersion;
		}
	},

	// 保存文档 - 节流保存到 Go API
	async onStoreDocument(data: any) {
		const document = data.document as Y.Doc;
		const context = data.context as RealtimeContext | undefined;
		const documentId = context?.documentId || normalizeDocumentId(data?.documentName);
		const userId = context?.userId;
		const token = context?.token;

		if (!documentId || !token || !userId || !context) {
			console.warn('Missing context for storing document');
			return;
		}

		// Re-validate ACL on every save. The captured-at-connect ACL is stale
		// the moment the document owner removes the editor. A 30s TTL keeps
		// the cost down without leaving a wide window for revoked users.
		const freshACL = await getUserACLFresh(documentId, userId, token);
		if (!freshACL?.canEdit) {
			console.warn(
				`[DOC:${documentId}] User ${userId} lost edit permission; refusing to persist (had role ${context.acl.myRole})`
			);
			// Update the cached context so subsequent reads also see the
			// downgraded permission, and bubble up so Hocuspocus knows the
			// store didn't succeed.
			if (freshACL) {
				context.acl = freshACL;
			}
			throw new Error('edit permission revoked');
		}
		// Keep the context in sync with the latest ACL view.
		context.acl = freshACL;

		const yjsState = Buffer.from(Y.encodeStateAsUpdate(document)).toString('base64');
		const yjsStateVector = Buffer.from(Y.encodeStateVector(document)).toString('base64');

		try {
			const newVersion = await saveYjsState(
				documentId,
				token,
				yjsState,
				yjsStateVector,
				context.yjsVersion
			);
			context.yjsVersion = newVersion;
		} catch (error) {
			if (error instanceof YjsSaveConflictError) {
				// Someone else (or a racing save) bumped the version. Re-load
				// the latest state, merge it into our in-memory doc via
				// Yjs CRDT semantics, and let Hocuspocus retry on the next
				// debounce. Throwing keeps the doc marked dirty so the retry
				// actually happens.
				console.warn(
					`[DOC:${documentId}] Yjs save conflict (had ${context.yjsVersion}, server ${error.currentVersion}); reloading`
				);
				const fresh = await loadYjsState(documentId, token);
				if (fresh.yjsState) {
					try {
						Y.applyUpdate(document, Buffer.from(fresh.yjsState, 'base64'));
					} catch (mergeErr) {
						console.error(`[DOC:${documentId}] Failed to merge fresh state:`, mergeErr);
					}
				}
				context.yjsVersion = fresh.yjsVersion;
			}
			throw error;
		}
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

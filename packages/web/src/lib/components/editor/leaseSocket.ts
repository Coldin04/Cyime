import { auth } from '$lib/stores/auth';
import { resolveApiUrl } from '$lib/config/api';

const RECONNECT_DELAY_MS = 3000;

type LeaseClaimedEvent = {
	type: 'lease_claimed';
	leaseToken: string;
};

function resolveEventsSocketUrl(documentId: string): string {
	const httpUrl = resolveApiUrl(`/api/v1/edit-lease/documents/${documentId}/events`);
	return httpUrl.replace(/^http/, 'ws');
}

/**
 * Subscribes to server-pushed lease-change notifications for a document over
 * a lightweight WebSocket the edit-lease backend exposes purely for this —
 * no document content ever crosses it, only "the lease token changed"
 * events. This is what lets a takeover on a *different device/browser*
 * invalidate the original session right away, instead of it waiting for its
 * next ~20s renewal poll to fail. `leaseBroadcast.ts` covers the same-browser
 * case (via BroadcastChannel); this covers everything else.
 *
 * The connection is a courtesy fast path, not a dependency: if it never
 * connects (blocked by a proxy, server restarting, browser tab throttled in
 * the background, etc.) or keeps dropping, nothing regresses — the existing
 * renewal poll remains the source of truth and keeps running independently
 * of this.
 *
 * The socket authenticates once, right after it opens, with the access token
 * current at connect time — there's no per-message re-auth for the
 * connection's lifetime. That's an intentional simplification: this channel
 * only ever tells a listener "your token is stale, go check," never anything
 * about document content, so re-validating a long-lived socket mid-flight
 * isn't worth the added complexity.
 */
export function subscribeToLeaseEvents(
	documentId: string,
	currentLeaseToken: string,
	onClaimedElsewhere: () => void
): () => void {
	let socket: WebSocket | null = null;
	let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
	let stopped = false;

	function scheduleReconnect() {
		if (stopped || reconnectTimer) return;
		reconnectTimer = setTimeout(() => {
			reconnectTimer = null;
			connect();
		}, RECONNECT_DELAY_MS);
	}

	function connect() {
		if (stopped) return;
		const accessToken = auth.getAccessToken();
		if (!accessToken) {
			// Not signed in (yet), e.g. a token refresh is still in flight —
			// try again shortly instead of giving up on the channel entirely.
			scheduleReconnect();
			return;
		}

		let ws: WebSocket;
		try {
			ws = new WebSocket(resolveEventsSocketUrl(documentId));
		} catch (error) {
			console.error('[LeaseEvents] Failed to open socket:', error);
			scheduleReconnect();
			return;
		}
		socket = ws;

		ws.addEventListener('open', () => {
			ws.send(JSON.stringify({ accessToken }));
		});

		ws.addEventListener('message', (event) => {
			try {
				const data = JSON.parse(event.data as string) as Partial<LeaseClaimedEvent>;
				if (
					data.type === 'lease_claimed' &&
					data.leaseToken &&
					data.leaseToken !== currentLeaseToken
				) {
					onClaimedElsewhere();
				}
			} catch (error) {
				console.error('[LeaseEvents] Failed to parse message:', error);
			}
		});

		ws.addEventListener('close', () => {
			if (socket === ws) socket = null;
			scheduleReconnect();
		});
		ws.addEventListener('error', () => ws.close());
	}

	connect();

	return () => {
		stopped = true;
		if (reconnectTimer) {
			clearTimeout(reconnectTimer);
			reconnectTimer = null;
		}
		socket?.close();
		socket = null;
	};
}

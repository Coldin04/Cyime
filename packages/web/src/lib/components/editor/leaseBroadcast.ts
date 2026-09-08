type LeaseClaimedMessage = {
	documentId: string;
	leaseToken: string;
};

const CHANNEL_NAME = 'cyime-edit-lease';

function openChannel(): BroadcastChannel | null {
	if (typeof BroadcastChannel === 'undefined') return null;
	try {
		return new BroadcastChannel(CHANNEL_NAME);
	} catch {
		return null;
	}
}

/**
 * Tells every other same-origin tab that this tab now holds the edit lease
 * for `documentId`. A BroadcastChannel never delivers a message back to its
 * own sender, so any tab that receives this and isn't the new holder can
 * treat its own lease as stale right away — instead of waiting for its next
 * ~20s renewal heartbeat to fail against the server.
 *
 * This is a same-browser courtesy signal only, not the source of truth: the
 * server-side lease (and the periodic renewal check) stays authoritative, so
 * a dropped or unsupported broadcast just falls back to the existing delay.
 */
export function broadcastLeaseClaimed(documentId: string, leaseToken: string): void {
	const channel = openChannel();
	if (!channel) return;
	try {
		channel.postMessage({ documentId, leaseToken } satisfies LeaseClaimedMessage);
	} finally {
		channel.close();
	}
}

/**
 * Calls `onClaimedElsewhere` the moment another same-origin tab claims (or
 * takes over) the edit lease for `documentId` with a token different from
 * `currentLeaseToken`. Returns an unsubscribe function.
 */
export function subscribeToLeaseClaimedElsewhere(
	documentId: string,
	currentLeaseToken: string,
	onClaimedElsewhere: () => void
): () => void {
	const channel = openChannel();
	if (!channel) return () => {};

	const handleMessage = (event: MessageEvent<LeaseClaimedMessage>) => {
		const message = event.data;
		if (!message || message.documentId !== documentId) return;
		if (message.leaseToken === currentLeaseToken) return;
		onClaimedElsewhere();
	};

	channel.addEventListener('message', handleMessage);
	return () => {
		channel.removeEventListener('message', handleMessage);
		channel.close();
	};
}

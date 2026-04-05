import { env } from '$env/dynamic/public';

const DEFAULT_AVATAR_MAX_BYTES = 2 * 1024 * 1024;
const DEFAULT_AVATAR_OUTPUT_SIZE = 512;

function parsePositiveInt(raw: string | undefined, fallback: number): number {
	if (!raw) return fallback;
	const parsed = Number.parseInt(raw, 10);
	if (!Number.isFinite(parsed) || parsed <= 0) return fallback;
	return parsed;
}

export const avatarMaxBytes = parsePositiveInt(env.PUBLIC_AVATAR_MAX_BYTES, DEFAULT_AVATAR_MAX_BYTES);
export const avatarOutputSize = parsePositiveInt(
	env.PUBLIC_AVATAR_OUTPUT_SIZE,
	DEFAULT_AVATAR_OUTPUT_SIZE
);

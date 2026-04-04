import { spawnSync } from 'node:child_process';

const explicitTarget = (process.env.DEPLOY_TARGET ?? '').trim().toLowerCase();
const inferredTarget = process.env.CF_PAGES === '1' ? 'cloudflare' : '';
const target = explicitTarget || inferredTarget;

const command =
	target === 'cloudflare'
		? ['pnpm', 'run', 'build:cloudflare']
		: ['pnpm', 'exec', 'vite', 'build'];

const result = spawnSync(command[0], command.slice(1), {
	stdio: 'inherit',
	shell: false,
	env: process.env
});

if (result.error) {
	throw result.error;
}

process.exit(result.status ?? 1);

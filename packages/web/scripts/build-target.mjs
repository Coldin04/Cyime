import { spawnSync } from 'node:child_process';

const cliTarget = (process.argv[2] ?? '').trim().toLowerCase();
const explicitTarget = (process.env.DEPLOY_TARGET ?? '').trim().toLowerCase();
const inferredTarget = process.env.CF_PAGES === '1' ? 'cloudflare' : '';
const target = cliTarget || explicitTarget || inferredTarget;

const pnpmBin = process.platform === 'win32' ? 'pnpm.cmd' : 'pnpm';
const command = [pnpmBin, 'exec', 'vite', 'build'];
const env = {
	...process.env,
	...(target ? { DEPLOY_TARGET: target } : {})
};

const result = spawnSync(command[0], command.slice(1), {
	stdio: 'inherit',
	shell: false,
	env
});

if (result.error) {
	throw result.error;
}

process.exit(result.status ?? 1);

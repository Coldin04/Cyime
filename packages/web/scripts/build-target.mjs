const cliTarget = (process.argv[2] ?? '').trim().toLowerCase();
const explicitTarget = (process.env.DEPLOY_TARGET ?? '').trim().toLowerCase();
const inferredTarget = process.env.CF_PAGES === '1' ? 'cloudflare' : '';
const target = cliTarget || explicitTarget || inferredTarget;

if (target) {
	process.env.DEPLOY_TARGET = target;
}

const { build } = await import('vite');
await build();

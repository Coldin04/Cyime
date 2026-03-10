<script>import '$lib/paraglide-strategy.ts';
import "../app.css";
import favicon from '$lib/assets/favicon.svg';
import { Toaster } from 'svelte-sonner';

let { children } = $props();</script>

<svelte:head>
	<link rel="icon" href="{favicon}" />
</svelte:head>

<Toaster richColors closeButton />
{@render children()}

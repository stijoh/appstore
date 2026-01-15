<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';

	let { children } = $props();

	const navLinks = [
		{ href: '/', label: 'Catalog', icon: 'grid' },
		{ href: '/deployments', label: 'Deployments', icon: 'layers' }
	];
</script>

<svelte:head>
	<title>App Store</title>
	<meta name="description" content="Kubernetes App Store - Deploy infrastructure apps with ease" />
</svelte:head>

<div class="app-shell">
	<nav class="sidebar">
		<div class="sidebar-header">
			<a href="/" class="logo">
				<div class="logo-icon">
					<svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M12 2L2 7l10 5 10-5-10-5z"/>
						<path d="M2 17l10 5 10-5"/>
						<path d="M2 12l10 5 10-5"/>
					</svg>
				</div>
				<span class="logo-text">AppStore</span>
			</a>
		</div>

		<div class="nav-section">
			<span class="nav-label">Menu</span>
			<ul class="nav-list">
				{#each navLinks as link}
					<li>
						<a
							href={link.href}
							class="nav-link"
							class:active={page.url.pathname === link.href ||
								(link.href !== '/' && page.url.pathname.startsWith(link.href))}
						>
							{#if link.icon === 'grid'}
								<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<rect x="3" y="3" width="7" height="7" rx="1"/>
									<rect x="14" y="3" width="7" height="7" rx="1"/>
									<rect x="3" y="14" width="7" height="7" rx="1"/>
									<rect x="14" y="14" width="7" height="7" rx="1"/>
								</svg>
							{:else if link.icon === 'layers'}
								<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<path d="M12 2L2 7l10 5 10-5-10-5z"/>
									<path d="M2 17l10 5 10-5"/>
									<path d="M2 12l10 5 10-5"/>
								</svg>
							{/if}
							<span>{link.label}</span>
						</a>
					</li>
				{/each}
			</ul>
		</div>

		<div class="sidebar-footer">
			<div class="version-badge">
				<span class="version-dot"></span>
				<span>v1.0.0</span>
			</div>
		</div>
	</nav>

	<main class="main-content">
		<div class="content-wrapper">
			{@render children()}
		</div>
	</main>
</div>

<style>
	.app-shell {
		display: flex;
		min-height: 100vh;
	}

	.sidebar {
		width: 260px;
		background: var(--bg-deep);
		border-right: 1px solid var(--border-subtle);
		display: flex;
		flex-direction: column;
		position: fixed;
		top: 0;
		left: 0;
		bottom: 0;
		z-index: 100;
	}

	.sidebar-header {
		padding: var(--space-lg);
		border-bottom: 1px solid var(--border-subtle);
	}

	.logo {
		display: flex;
		align-items: center;
		gap: var(--space-md);
		color: var(--text-primary);
		text-decoration: none;
	}

	.logo-icon {
		width: 44px;
		height: 44px;
		background: linear-gradient(135deg, var(--lavender-600), var(--lavender-800));
		border-radius: var(--radius-lg);
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
		box-shadow: 0 4px 16px rgba(147, 51, 234, 0.3);
	}

	.logo-text {
		font-size: 1.25rem;
		font-weight: 600;
		letter-spacing: -0.02em;
	}

	.nav-section {
		padding: var(--space-lg);
		flex: 1;
	}

	.nav-label {
		display: block;
		font-size: 0.75rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--text-muted);
		margin-bottom: var(--space-md);
	}

	.nav-list {
		list-style: none;
		display: flex;
		flex-direction: column;
		gap: var(--space-xs);
	}

	.nav-link {
		display: flex;
		align-items: center;
		gap: var(--space-md);
		padding: 0.75rem var(--space-md);
		border-radius: var(--radius-md);
		color: var(--text-secondary);
		text-decoration: none;
		font-weight: 500;
		transition: all var(--transition-fast);
		position: relative;
	}

	.nav-link:hover {
		background: var(--bg-surface);
		color: var(--text-primary);
	}

	.nav-link.active {
		background: var(--accent-glow);
		color: var(--accent);
	}

	.nav-link.active::before {
		content: '';
		position: absolute;
		left: 0;
		top: 50%;
		transform: translateY(-50%);
		width: 3px;
		height: 24px;
		background: var(--accent);
		border-radius: 0 2px 2px 0;
	}

	.nav-link svg {
		flex-shrink: 0;
		opacity: 0.7;
	}

	.nav-link.active svg,
	.nav-link:hover svg {
		opacity: 1;
	}

	.sidebar-footer {
		padding: var(--space-lg);
		border-top: 1px solid var(--border-subtle);
	}

	.version-badge {
		display: flex;
		align-items: center;
		gap: var(--space-sm);
		font-size: 0.8125rem;
		color: var(--text-muted);
		font-family: var(--font-mono);
	}

	.version-dot {
		width: 8px;
		height: 8px;
		background: var(--success);
		border-radius: 50%;
		box-shadow: 0 0 8px var(--success);
	}

	.main-content {
		flex: 1;
		margin-left: 260px;
		position: relative;
		z-index: 1;
	}

	.content-wrapper {
		padding: var(--space-2xl);
		max-width: 1400px;
		margin: 0 auto;
	}

	@media (max-width: 1024px) {
		.sidebar {
			width: 80px;
		}

		.logo-text,
		.nav-label,
		.nav-link span,
		.version-badge span:last-child {
			display: none;
		}

		.nav-link {
			justify-content: center;
			padding: 0.75rem;
		}

		.nav-link.active::before {
			display: none;
		}

		.main-content {
			margin-left: 80px;
		}
	}
</style>

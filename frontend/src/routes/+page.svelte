<script lang="ts">
	import { api, type App } from '$lib/api/client';
	import { onMount } from 'svelte';

	let apps = $state<App[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let searchQuery = $state('');
	let selectedCategory = $state<string | null>(null);

	const categories = $derived(() => {
		const cats = new Set(apps.map(app => app.category));
		return Array.from(cats).sort();
	});

	const filteredApps = $derived(() => {
		return apps.filter(app => {
			const matchesSearch = !searchQuery ||
				app.displayName.toLowerCase().includes(searchQuery.toLowerCase()) ||
				app.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
				app.tags.some(tag => tag.toLowerCase().includes(searchQuery.toLowerCase()));
			const matchesCategory = !selectedCategory || app.category === selectedCategory;
			return matchesSearch && matchesCategory;
		});
	});

	onMount(async () => {
		try {
			const response = await api.getCatalog();
			apps = response.apps || [];
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load catalog';
		} finally {
			loading = false;
		}
	});

	function getCategoryIcon(category: string): string {
		const icons: Record<string, string> = {
			database: '🗄️',
			cache: '⚡',
			messaging: '📨',
			storage: '💾',
			security: '🔐',
			monitoring: '📊'
		};
		return icons[category] || '📦';
	}
</script>

<div class="catalog-page">
	<header class="page-header">
		<div class="header-content">
			<h1 class="page-title">App Catalog</h1>
			<p class="page-description">
				Deploy production-ready infrastructure applications with a single click
			</p>
		</div>

		<div class="search-container">
			<div class="search-wrapper">
				<svg class="search-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<circle cx="11" cy="11" r="8"/>
					<path d="M21 21l-4.35-4.35"/>
				</svg>
				<input
					type="text"
					placeholder="Search apps..."
					bind:value={searchQuery}
					class="search-input"
				/>
				{#if searchQuery}
					<button class="search-clear" onclick={() => searchQuery = ''} aria-label="Clear search">
						<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M18 6L6 18M6 6l12 12"/>
						</svg>
					</button>
				{/if}
			</div>
		</div>
	</header>

	{#if loading}
		<div class="loading-grid">
			{#each Array(6) as _}
				<div class="skeleton-card">
					<div class="skeleton skeleton-icon"></div>
					<div class="skeleton skeleton-title"></div>
					<div class="skeleton skeleton-text"></div>
					<div class="skeleton skeleton-text short"></div>
				</div>
			{/each}
		</div>
	{:else if error}
		<div class="error-state">
			<div class="error-icon">
				<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<circle cx="12" cy="12" r="10"/>
					<path d="M12 8v4M12 16h.01"/>
				</svg>
			</div>
			<h3>Failed to load catalog</h3>
			<p>{error}</p>
			<button class="btn-primary" onclick={() => location.reload()}>
				Try Again
			</button>
		</div>
	{:else}
		<div class="filters">
			<button
				class="filter-chip"
				class:active={!selectedCategory}
				onclick={() => selectedCategory = null}
			>
				All
			</button>
			{#each categories() as category}
				<button
					class="filter-chip"
					class:active={selectedCategory === category}
					onclick={() => selectedCategory = category}
				>
					<span class="chip-icon">{getCategoryIcon(category)}</span>
					{category}
				</button>
			{/each}
		</div>

		{#if filteredApps().length === 0}
			<div class="empty-state">
				<div class="empty-icon">
					<svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
						<rect x="3" y="3" width="7" height="7" rx="1"/>
						<rect x="14" y="3" width="7" height="7" rx="1"/>
						<rect x="3" y="14" width="7" height="7" rx="1"/>
						<rect x="14" y="14" width="7" height="7" rx="1"/>
					</svg>
				</div>
				<h3>No apps found</h3>
				<p>Try adjusting your search or filters</p>
			</div>
		{:else}
			<div class="app-grid">
				{#each filteredApps() as app, i}
					<a href="/apps/{app.name}" class="app-card" style="--delay: {i * 50}ms">
						<div class="card-glow"></div>
						<div class="card-content">
							<div class="app-header">
								<div class="app-icon" data-category={app.category}>
									{getCategoryIcon(app.category)}
								</div>
								<span class="app-category">{app.category}</span>
							</div>

							<h3 class="app-name">{app.displayName}</h3>
							<p class="app-description">{app.description}</p>

							<div class="app-tags">
								{#each app.tags.slice(0, 3) as tag}
									<span class="tag">{tag}</span>
								{/each}
							</div>

							<div class="card-footer">
								<span class="deploy-label">Deploy</span>
								<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<path d="M5 12h14M12 5l7 7-7 7"/>
								</svg>
							</div>
						</div>
					</a>
				{/each}
			</div>
		{/if}
	{/if}
</div>

<style>
	.catalog-page {
		animation: fadeIn var(--transition-base) ease-out;
	}

	.page-header {
		margin-bottom: var(--space-2xl);
	}

	.header-content {
		margin-bottom: var(--space-xl);
	}

	.page-title {
		font-size: 2.5rem;
		font-weight: 700;
		letter-spacing: -0.03em;
		margin-bottom: var(--space-sm);
		background: linear-gradient(135deg, var(--text-primary) 0%, var(--lavender-300) 100%);
		-webkit-background-clip: text;
		-webkit-text-fill-color: transparent;
		background-clip: text;
	}

	.page-description {
		font-size: 1.125rem;
		color: var(--text-muted);
		max-width: 500px;
	}

	.search-container {
		max-width: 480px;
	}

	.search-wrapper {
		position: relative;
	}

	.search-icon {
		position: absolute;
		left: 1rem;
		top: 50%;
		transform: translateY(-50%);
		color: var(--text-muted);
		pointer-events: none;
	}

	.search-input {
		width: 100%;
		padding-left: 3rem;
		padding-right: 2.5rem;
		background: var(--bg-elevated);
		border: 1px solid var(--border-default);
		font-size: 1rem;
	}

	.search-input:focus {
		border-color: var(--accent);
		box-shadow: 0 0 0 3px var(--accent-glow), var(--shadow-glow);
	}

	.search-clear {
		position: absolute;
		right: 0.75rem;
		top: 50%;
		transform: translateY(-50%);
		background: none;
		border: none;
		padding: 0.25rem;
		color: var(--text-muted);
		cursor: pointer;
		border-radius: var(--radius-sm);
	}

	.search-clear:hover {
		color: var(--text-primary);
		background: var(--bg-surface);
	}

	.filters {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-sm);
		margin-bottom: var(--space-xl);
	}

	.filter-chip {
		display: inline-flex;
		align-items: center;
		gap: var(--space-xs);
		padding: 0.5rem 1rem;
		background: var(--bg-elevated);
		border: 1px solid var(--border-subtle);
		border-radius: var(--radius-xl);
		color: var(--text-secondary);
		font-size: 0.875rem;
		font-weight: 500;
		cursor: pointer;
		transition: all var(--transition-fast);
		text-transform: capitalize;
	}

	.filter-chip:hover {
		background: var(--bg-surface);
		border-color: var(--border-default);
		color: var(--text-primary);
	}

	.filter-chip.active {
		background: var(--accent-glow);
		border-color: var(--accent);
		color: var(--accent);
	}

	.chip-icon {
		font-size: 1rem;
	}

	.app-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
		gap: var(--space-lg);
	}

	.app-card {
		position: relative;
		background: var(--bg-elevated);
		border: 1px solid var(--border-subtle);
		border-radius: var(--radius-xl);
		text-decoration: none;
		overflow: hidden;
		transition: all var(--transition-base);
		animation: slideUp var(--transition-slow) ease-out backwards;
		animation-delay: var(--delay);
	}

	.card-glow {
		position: absolute;
		inset: 0;
		background: radial-gradient(
			circle at 50% 0%,
			rgba(147, 51, 234, 0.08) 0%,
			transparent 60%
		);
		opacity: 0;
		transition: opacity var(--transition-base);
	}

	.app-card:hover {
		border-color: var(--border-accent);
		transform: translateY(-4px);
		box-shadow: var(--shadow-lg), 0 0 40px rgba(147, 51, 234, 0.1);
	}

	.app-card:hover .card-glow {
		opacity: 1;
	}

	.card-content {
		position: relative;
		padding: var(--space-xl);
	}

	.app-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: var(--space-lg);
	}

	.app-icon {
		width: 56px;
		height: 56px;
		background: var(--bg-surface);
		border-radius: var(--radius-lg);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1.75rem;
		border: 1px solid var(--border-subtle);
	}

	.app-icon[data-category="database"] {
		background: linear-gradient(135deg, rgba(59, 130, 246, 0.15), rgba(59, 130, 246, 0.05));
		border-color: rgba(59, 130, 246, 0.2);
	}

	.app-icon[data-category="cache"] {
		background: linear-gradient(135deg, rgba(234, 179, 8, 0.15), rgba(234, 179, 8, 0.05));
		border-color: rgba(234, 179, 8, 0.2);
	}

	.app-category {
		font-size: 0.75rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--text-muted);
		background: var(--bg-surface);
		padding: 0.25rem 0.75rem;
		border-radius: var(--radius-xl);
	}

	.app-name {
		font-size: 1.25rem;
		font-weight: 600;
		color: var(--text-primary);
		margin-bottom: var(--space-sm);
	}

	.app-description {
		font-size: 0.9375rem;
		color: var(--text-secondary);
		line-height: 1.6;
		margin-bottom: var(--space-lg);
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.app-tags {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-xs);
		margin-bottom: var(--space-lg);
	}

	.tag {
		font-size: 0.75rem;
		font-family: var(--font-mono);
		color: var(--lavender-300);
		background: rgba(147, 51, 234, 0.1);
		padding: 0.25rem 0.625rem;
		border-radius: var(--radius-sm);
	}

	.card-footer {
		display: flex;
		align-items: center;
		gap: var(--space-sm);
		color: var(--accent);
		font-size: 0.875rem;
		font-weight: 500;
		opacity: 0;
		transform: translateX(-8px);
		transition: all var(--transition-fast);
	}

	.app-card:hover .card-footer {
		opacity: 1;
		transform: translateX(0);
	}

	.card-footer svg {
		transition: transform var(--transition-fast);
	}

	.app-card:hover .card-footer svg {
		transform: translateX(4px);
	}

	/* Loading state */
	.loading-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
		gap: var(--space-lg);
	}

	.skeleton-card {
		background: var(--bg-elevated);
		border: 1px solid var(--border-subtle);
		border-radius: var(--radius-xl);
		padding: var(--space-xl);
	}

	.skeleton-icon {
		width: 56px;
		height: 56px;
		border-radius: var(--radius-lg);
		margin-bottom: var(--space-lg);
	}

	.skeleton-title {
		height: 24px;
		width: 60%;
		margin-bottom: var(--space-md);
	}

	.skeleton-text {
		height: 16px;
		width: 100%;
		margin-bottom: var(--space-sm);
	}

	.skeleton-text.short {
		width: 40%;
	}

	/* Empty & Error states */
	.empty-state,
	.error-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: var(--space-3xl);
		text-align: center;
	}

	.empty-icon,
	.error-icon {
		color: var(--text-muted);
		margin-bottom: var(--space-lg);
		opacity: 0.5;
	}

	.error-icon {
		color: var(--error);
	}

	.empty-state h3,
	.error-state h3 {
		color: var(--text-primary);
		margin-bottom: var(--space-sm);
	}

	.empty-state p,
	.error-state p {
		color: var(--text-muted);
		margin-bottom: var(--space-lg);
	}

	@media (max-width: 768px) {
		.page-title {
			font-size: 2rem;
		}

		.app-grid {
			grid-template-columns: 1fr;
		}
	}
</style>

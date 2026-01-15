<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, type App } from '$lib/api/client';
	import { onMount } from 'svelte';

	let app = $state<App | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let deploying = $state(false);
	let deployError = $state<string | null>(null);
	let deploySuccess = $state(false);

	// Form state
	let releaseName = $state('');
	let namespace = $state('default');
	let valuesYaml = $state('');

	const appName = $derived(page.params.appName as string);

	onMount(async () => {
		if (!appName) return;
		try {
			app = await api.getApp(appName);
			releaseName = `my-${app.name}`;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load app';
		} finally {
			loading = false;
		}
	});

	async function handleDeploy() {
		if (!app) return;

		deploying = true;
		deployError = null;
		deploySuccess = false;

		try {
			// Parse YAML values if provided
			let values: Record<string, unknown> | undefined;
			if (valuesYaml.trim()) {
				// Simple YAML parsing for key: value pairs
				values = {};
				const lines = valuesYaml.split('\n');
				for (const line of lines) {
					const match = line.match(/^(\w+):\s*(.+)$/);
					if (match) {
						const [, key, value] = match;
						// Try to parse as number or boolean
						if (value === 'true') values[key] = true;
						else if (value === 'false') values[key] = false;
						else if (!isNaN(Number(value))) values[key] = Number(value);
						else values[key] = value.replace(/^["']|["']$/g, '');
					}
				}
			}

			await api.createDeployment({
				appName: app.name,
				namespace,
				releaseName: releaseName || undefined,
				values
			});

			deploySuccess = true;
			setTimeout(() => {
				goto('/deployments');
			}, 2000);
		} catch (e) {
			deployError = e instanceof Error ? e.message : 'Deployment failed';
		} finally {
			deploying = false;
		}
	}

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

<div class="app-detail-page">
	<a href="/" class="back-link">
		<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<path d="M19 12H5M12 19l-7-7 7-7"/>
		</svg>
		<span>Back to Catalog</span>
	</a>

	{#if loading}
		<div class="loading-state">
			<div class="skeleton skeleton-header"></div>
			<div class="skeleton skeleton-content"></div>
		</div>
	{:else if error}
		<div class="error-state">
			<div class="error-icon">
				<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<circle cx="12" cy="12" r="10"/>
					<path d="M12 8v4M12 16h.01"/>
				</svg>
			</div>
			<h3>App not found</h3>
			<p>{error}</p>
			<a href="/" class="btn-primary">Browse Catalog</a>
		</div>
	{:else if app}
		<div class="content-grid">
			<div class="app-info">
				<header class="app-header">
					<div class="app-icon" data-category={app.category}>
						{getCategoryIcon(app.category)}
					</div>
					<div class="app-meta">
						<span class="app-category">{app.category}</span>
						<h1 class="app-name">{app.displayName}</h1>
					</div>
				</header>

				<p class="app-description">{app.description}</p>

				<div class="app-tags">
					{#each app.tags as tag}
						<span class="tag">{tag}</span>
					{/each}
				</div>

				<div class="info-section">
					<h3>About this app</h3>
					<div class="info-grid">
						<div class="info-item">
							<span class="info-label">Chart Path</span>
							<span class="info-value font-mono">{app.chartPath}</span>
						</div>
						<div class="info-item">
							<span class="info-label">Category</span>
							<span class="info-value">{app.category}</span>
						</div>
					</div>
				</div>
			</div>

			<div class="deploy-panel">
				<div class="panel-header">
					<h2>Deploy {app.displayName}</h2>
					<p>Configure your deployment settings</p>
				</div>

				{#if deploySuccess}
					<div class="success-message">
						<div class="success-icon">
							<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M20 6L9 17l-5-5"/>
							</svg>
						</div>
						<div>
							<h4>Deployment Initiated!</h4>
							<p>Redirecting to deployments...</p>
						</div>
					</div>
				{:else}
					<form onsubmit={(e) => { e.preventDefault(); handleDeploy(); }} class="deploy-form">
						<div class="form-group">
							<label for="releaseName">Release Name</label>
							<input
								id="releaseName"
								type="text"
								bind:value={releaseName}
								placeholder="my-{app.name}"
								disabled={deploying}
							/>
							<span class="form-hint">Unique name for this deployment</span>
						</div>

						<div class="form-group">
							<label for="namespace">Namespace</label>
							<input
								id="namespace"
								type="text"
								bind:value={namespace}
								placeholder="default"
								disabled={deploying}
							/>
							<span class="form-hint">Kubernetes namespace to deploy to</span>
						</div>

						<div class="form-group">
							<label for="values">Custom Values <span class="optional">(optional)</span></label>
							<textarea
								id="values"
								bind:value={valuesYaml}
								placeholder="# YAML format&#10;key: value&#10;replicas: 1"
								rows="6"
								disabled={deploying}
							></textarea>
							<span class="form-hint">Override default Helm chart values</span>
						</div>

						{#if deployError}
							<div class="error-message">
								<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<circle cx="12" cy="12" r="10"/>
									<path d="M12 8v4M12 16h.01"/>
								</svg>
								<span>{deployError}</span>
							</div>
						{/if}

						<button type="submit" class="btn-primary deploy-btn" disabled={deploying}>
							{#if deploying}
								<span class="spinner"></span>
								<span>Deploying...</span>
							{:else}
								<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<path d="M12 2L2 7l10 5 10-5-10-5z"/>
									<path d="M2 17l10 5 10-5"/>
									<path d="M2 12l10 5 10-5"/>
								</svg>
								<span>Deploy</span>
							{/if}
						</button>
					</form>
				{/if}
			</div>
		</div>
	{/if}
</div>

<style>
	.app-detail-page {
		animation: fadeIn var(--transition-base) ease-out;
	}

	.back-link {
		display: inline-flex;
		align-items: center;
		gap: var(--space-sm);
		color: var(--text-muted);
		text-decoration: none;
		font-size: 0.9375rem;
		margin-bottom: var(--space-xl);
		transition: color var(--transition-fast);
	}

	.back-link:hover {
		color: var(--text-primary);
	}

	.back-link svg {
		transition: transform var(--transition-fast);
	}

	.back-link:hover svg {
		transform: translateX(-4px);
	}

	.content-grid {
		display: grid;
		grid-template-columns: 1fr 420px;
		gap: var(--space-2xl);
		align-items: start;
	}

	.app-info {
		animation: slideUp var(--transition-slow) ease-out;
	}

	.app-header {
		display: flex;
		align-items: flex-start;
		gap: var(--space-lg);
		margin-bottom: var(--space-xl);
	}

	.app-icon {
		width: 80px;
		height: 80px;
		background: var(--bg-elevated);
		border-radius: var(--radius-xl);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 2.5rem;
		border: 1px solid var(--border-subtle);
		flex-shrink: 0;
	}

	.app-icon[data-category="database"] {
		background: linear-gradient(135deg, rgba(59, 130, 246, 0.15), rgba(59, 130, 246, 0.05));
		border-color: rgba(59, 130, 246, 0.2);
	}

	.app-icon[data-category="cache"] {
		background: linear-gradient(135deg, rgba(234, 179, 8, 0.15), rgba(234, 179, 8, 0.05));
		border-color: rgba(234, 179, 8, 0.2);
	}

	.app-meta {
		flex: 1;
	}

	.app-category {
		display: inline-block;
		font-size: 0.75rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--lavender-400);
		margin-bottom: var(--space-xs);
	}

	.app-name {
		font-size: 2rem;
		font-weight: 700;
		letter-spacing: -0.02em;
		color: var(--text-primary);
	}

	.app-description {
		font-size: 1.125rem;
		color: var(--text-secondary);
		line-height: 1.7;
		margin-bottom: var(--space-xl);
	}

	.app-tags {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-sm);
		margin-bottom: var(--space-2xl);
	}

	.tag {
		font-size: 0.8125rem;
		font-family: var(--font-mono);
		color: var(--lavender-300);
		background: rgba(147, 51, 234, 0.1);
		padding: 0.375rem 0.875rem;
		border-radius: var(--radius-md);
		border: 1px solid rgba(147, 51, 234, 0.2);
	}

	.info-section {
		background: var(--bg-elevated);
		border: 1px solid var(--border-subtle);
		border-radius: var(--radius-xl);
		padding: var(--space-xl);
	}

	.info-section h3 {
		font-size: 1rem;
		font-weight: 600;
		color: var(--text-primary);
		margin-bottom: var(--space-lg);
	}

	.info-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: var(--space-lg);
	}

	.info-item {
		display: flex;
		flex-direction: column;
		gap: var(--space-xs);
	}

	.info-label {
		font-size: 0.75rem;
		font-weight: 500;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--text-muted);
	}

	.info-value {
		font-size: 0.9375rem;
		color: var(--text-primary);
	}

	.font-mono {
		font-family: var(--font-mono);
	}

	/* Deploy Panel */
	.deploy-panel {
		background: var(--bg-elevated);
		border: 1px solid var(--border-subtle);
		border-radius: var(--radius-xl);
		padding: var(--space-xl);
		position: sticky;
		top: var(--space-xl);
		animation: slideUp var(--transition-slow) ease-out 100ms backwards;
	}

	.panel-header {
		margin-bottom: var(--space-xl);
		padding-bottom: var(--space-lg);
		border-bottom: 1px solid var(--border-subtle);
	}

	.panel-header h2 {
		font-size: 1.25rem;
		font-weight: 600;
		color: var(--text-primary);
		margin-bottom: var(--space-xs);
	}

	.panel-header p {
		font-size: 0.9375rem;
		color: var(--text-muted);
	}

	.deploy-form {
		display: flex;
		flex-direction: column;
		gap: var(--space-lg);
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: var(--space-xs);
	}

	.form-group label {
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--text-secondary);
	}

	.optional {
		font-weight: 400;
		color: var(--text-muted);
	}

	.form-group input,
	.form-group textarea {
		background: var(--bg-surface);
		border: 1px solid var(--border-default);
	}

	.form-group textarea {
		font-family: var(--font-mono);
		font-size: 0.875rem;
		resize: vertical;
		min-height: 120px;
	}

	.form-hint {
		font-size: 0.75rem;
		color: var(--text-muted);
	}

	.error-message {
		display: flex;
		align-items: center;
		gap: var(--space-sm);
		padding: var(--space-md);
		background: var(--error-muted);
		border: 1px solid rgba(248, 113, 113, 0.3);
		border-radius: var(--radius-md);
		color: var(--error);
		font-size: 0.875rem;
	}

	.success-message {
		display: flex;
		align-items: center;
		gap: var(--space-md);
		padding: var(--space-lg);
		background: var(--success-muted);
		border: 1px solid rgba(74, 222, 128, 0.3);
		border-radius: var(--radius-lg);
	}

	.success-icon {
		width: 48px;
		height: 48px;
		background: var(--success);
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--bg-deepest);
		flex-shrink: 0;
	}

	.success-message h4 {
		font-size: 1rem;
		font-weight: 600;
		color: var(--success);
		margin-bottom: var(--space-xs);
	}

	.success-message p {
		font-size: 0.875rem;
		color: var(--text-secondary);
	}

	.deploy-btn {
		width: 100%;
		padding: 1rem;
		font-size: 1rem;
		margin-top: var(--space-sm);
	}

	.spinner {
		width: 18px;
		height: 18px;
		border: 2px solid rgba(255, 255, 255, 0.3);
		border-top-color: white;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	/* Loading & Error states */
	.loading-state {
		display: flex;
		flex-direction: column;
		gap: var(--space-lg);
	}

	.skeleton-header {
		height: 100px;
		width: 100%;
	}

	.skeleton-content {
		height: 300px;
		width: 100%;
	}

	.error-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: var(--space-3xl);
		text-align: center;
	}

	.error-icon {
		color: var(--error);
		margin-bottom: var(--space-lg);
	}

	.error-state h3 {
		color: var(--text-primary);
		margin-bottom: var(--space-sm);
	}

	.error-state p {
		color: var(--text-muted);
		margin-bottom: var(--space-lg);
	}

	@media (max-width: 1024px) {
		.content-grid {
			grid-template-columns: 1fr;
		}

		.deploy-panel {
			position: static;
		}
	}

	@media (max-width: 768px) {
		.app-header {
			flex-direction: column;
			align-items: flex-start;
		}

		.app-name {
			font-size: 1.5rem;
		}

		.info-grid {
			grid-template-columns: 1fr;
		}
	}
</style>

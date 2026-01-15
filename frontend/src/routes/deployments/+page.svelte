<script lang="ts">
	import { api, type Deployment } from '$lib/api/client';
	import { onMount } from 'svelte';

	let deployments = $state<Deployment[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let deletingId = $state<string | null>(null);

	onMount(async () => {
		await loadDeployments();
	});

	async function loadDeployments() {
		try {
			const response = await api.getDeployments();
			deployments = response.deployments || [];
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load deployments';
		} finally {
			loading = false;
		}
	}

	async function handleDelete(deployment: Deployment) {
		if (!confirm(`Delete deployment "${deployment.name}"? This action cannot be undone.`)) {
			return;
		}

		deletingId = deployment.name;
		try {
			await api.deleteDeployment(deployment.name, deployment.namespace);
			// Remove from list immediately
			deployments = deployments.filter(d => d.name !== deployment.name);
		} catch (e) {
			alert(e instanceof Error ? e.message : 'Delete failed');
		} finally {
			deletingId = null;
		}
	}

	function getStatusColor(phase: string): string {
		switch (phase.toLowerCase()) {
			case 'deployed':
			case 'ready':
				return 'success';
			case 'installing':
			case 'upgrading':
			case 'pending':
				return 'warning';
			case 'failed':
			case 'error':
				return 'error';
			default:
				return 'info';
		}
	}

	function formatDate(dateString: string): string {
		const date = new Date(dateString);
		return date.toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}

	function getRelativeTime(dateString: string): string {
		const date = new Date(dateString);
		const now = new Date();
		const diffMs = now.getTime() - date.getTime();
		const diffMins = Math.floor(diffMs / 60000);
		const diffHours = Math.floor(diffMs / 3600000);
		const diffDays = Math.floor(diffMs / 86400000);

		if (diffMins < 1) return 'just now';
		if (diffMins < 60) return `${diffMins}m ago`;
		if (diffHours < 24) return `${diffHours}h ago`;
		if (diffDays < 7) return `${diffDays}d ago`;
		return formatDate(dateString);
	}
</script>

<div class="deployments-page">
	<header class="page-header">
		<div class="header-content">
			<h1 class="page-title">Deployments</h1>
			<p class="page-description">
				Manage your running applications
			</p>
		</div>
		<a href="/" class="btn-primary">
			<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M12 5v14M5 12h14"/>
			</svg>
			<span>New Deployment</span>
		</a>
	</header>

	{#if loading}
		<div class="loading-list">
			{#each Array(3) as _}
				<div class="skeleton-row">
					<div class="skeleton skeleton-icon"></div>
					<div class="skeleton-content">
						<div class="skeleton skeleton-title"></div>
						<div class="skeleton skeleton-text"></div>
					</div>
					<div class="skeleton skeleton-badge"></div>
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
			<h3>Failed to load deployments</h3>
			<p>{error}</p>
			<button class="btn-primary" onclick={() => { loading = true; error = null; loadDeployments(); }}>
				Try Again
			</button>
		</div>
	{:else if deployments.length === 0}
		<div class="empty-state">
			<div class="empty-illustration">
				<svg width="120" height="120" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="0.5">
					<path d="M12 2L2 7l10 5 10-5-10-5z"/>
					<path d="M2 17l10 5 10-5"/>
					<path d="M2 12l10 5 10-5"/>
				</svg>
			</div>
			<h3>No deployments yet</h3>
			<p>Deploy your first application from the catalog</p>
			<a href="/" class="btn-primary">
				Browse Catalog
			</a>
		</div>
	{:else}
		<div class="deployments-list">
			{#each deployments as deployment, i}
				<div class="deployment-card" style="--delay: {i * 50}ms">
					<div class="deployment-main">
						<div class="deployment-icon" data-app={deployment.appName}>
							<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M12 2L2 7l10 5 10-5-10-5z"/>
								<path d="M2 17l10 5 10-5"/>
								<path d="M2 12l10 5 10-5"/>
							</svg>
						</div>

						<div class="deployment-info">
							<div class="deployment-header">
								<h3 class="deployment-name">{deployment.name}</h3>
								<span class="status-badge" data-status={getStatusColor(deployment.phase)}>
									<span class="status-dot"></span>
									{deployment.phase}
								</span>
							</div>

							<div class="deployment-meta">
								<span class="meta-item">
									<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
										<rect x="3" y="3" width="7" height="7" rx="1"/>
										<rect x="14" y="3" width="7" height="7" rx="1"/>
										<rect x="3" y="14" width="7" height="7" rx="1"/>
										<rect x="14" y="14" width="7" height="7" rx="1"/>
									</svg>
									{deployment.appName}
								</span>
								<span class="meta-item">
									<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
										<path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
									</svg>
									{deployment.namespace}
								</span>
								<span class="meta-item version">
									v{deployment.deployedChartVersion || '—'}
								</span>
							</div>
						</div>
					</div>

					<div class="deployment-details">
						<div class="detail-item">
							<span class="detail-label">Helm Release</span>
							<span class="detail-value font-mono">{deployment.helmReleaseName}</span>
						</div>
						<div class="detail-item">
							<span class="detail-label">Revision</span>
							<span class="detail-value">#{deployment.helmReleaseRevision}</span>
						</div>
						<div class="detail-item">
							<span class="detail-label">Created</span>
							<span class="detail-value" title={formatDate(deployment.createdAt)}>
								{getRelativeTime(deployment.createdAt)}
							</span>
						</div>
						<div class="detail-item">
							<span class="detail-label">Last Sync</span>
							<span class="detail-value" title={formatDate(deployment.lastReconcileTime)}>
								{getRelativeTime(deployment.lastReconcileTime)}
							</span>
						</div>
					</div>

					{#if deployment.conditions && deployment.conditions.length > 0}
						<div class="conditions">
							{#each deployment.conditions.slice(0, 2) as condition}
								<div class="condition" data-status={condition.status === 'True' ? 'success' : 'muted'}>
									<span class="condition-type">{condition.type}</span>
									<span class="condition-message">{condition.message}</span>
								</div>
							{/each}
						</div>
					{/if}

					<div class="deployment-actions">
						<button
							class="btn-ghost btn-icon"
							title="View logs"
							disabled
						>
							<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
								<path d="M14 2v6h6"/>
								<path d="M16 13H8M16 17H8M10 9H8"/>
							</svg>
						</button>
						<button
							class="btn-ghost btn-icon delete-btn"
							title="Delete deployment"
							onclick={() => handleDelete(deployment)}
							disabled={deletingId === deployment.name}
						>
							{#if deletingId === deployment.name}
								<span class="spinner-small"></span>
							{:else}
								<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
								</svg>
							{/if}
						</button>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.deployments-page {
		animation: fadeIn var(--transition-base) ease-out;
	}

	.page-header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: var(--space-lg);
		margin-bottom: var(--space-2xl);
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
	}

	.deployments-list {
		display: flex;
		flex-direction: column;
		gap: var(--space-md);
	}

	.deployment-card {
		background: var(--bg-elevated);
		border: 1px solid var(--border-subtle);
		border-radius: var(--radius-xl);
		padding: var(--space-xl);
		display: grid;
		grid-template-columns: 1fr auto auto;
		gap: var(--space-xl);
		align-items: center;
		transition: all var(--transition-fast);
		animation: slideUp var(--transition-slow) ease-out backwards;
		animation-delay: var(--delay);
	}

	.deployment-card:hover {
		border-color: var(--border-default);
		background: var(--bg-surface);
	}

	.deployment-main {
		display: flex;
		align-items: center;
		gap: var(--space-lg);
	}

	.deployment-icon {
		width: 48px;
		height: 48px;
		background: var(--accent-glow);
		border-radius: var(--radius-lg);
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--accent);
		flex-shrink: 0;
	}

	.deployment-header {
		display: flex;
		align-items: center;
		gap: var(--space-md);
		margin-bottom: var(--space-xs);
	}

	.deployment-name {
		font-size: 1.125rem;
		font-weight: 600;
		color: var(--text-primary);
	}

	.status-badge {
		display: inline-flex;
		align-items: center;
		gap: var(--space-xs);
		font-size: 0.75rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.03em;
		padding: 0.25rem 0.625rem;
		border-radius: var(--radius-xl);
	}

	.status-badge[data-status="success"] {
		background: var(--success-muted);
		color: var(--success);
	}

	.status-badge[data-status="warning"] {
		background: var(--warning-muted);
		color: var(--warning);
	}

	.status-badge[data-status="error"] {
		background: var(--error-muted);
		color: var(--error);
	}

	.status-badge[data-status="info"] {
		background: var(--info-muted);
		color: var(--info);
	}

	.status-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: currentColor;
	}

	.deployment-meta {
		display: flex;
		align-items: center;
		gap: var(--space-lg);
	}

	.meta-item {
		display: flex;
		align-items: center;
		gap: var(--space-xs);
		font-size: 0.8125rem;
		color: var(--text-muted);
	}

	.meta-item.version {
		font-family: var(--font-mono);
		color: var(--lavender-400);
	}

	.deployment-details {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: var(--space-sm) var(--space-xl);
	}

	.detail-item {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.detail-label {
		font-size: 0.6875rem;
		font-weight: 500;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--text-muted);
	}

	.detail-value {
		font-size: 0.8125rem;
		color: var(--text-secondary);
	}

	.font-mono {
		font-family: var(--font-mono);
	}

	.conditions {
		grid-column: 1 / -1;
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-sm);
		padding-top: var(--space-md);
		border-top: 1px solid var(--border-subtle);
	}

	.condition {
		display: flex;
		align-items: center;
		gap: var(--space-sm);
		font-size: 0.75rem;
		padding: var(--space-xs) var(--space-sm);
		background: var(--bg-surface);
		border-radius: var(--radius-md);
	}

	.condition[data-status="success"] .condition-type {
		color: var(--success);
	}

	.condition[data-status="muted"] .condition-type {
		color: var(--text-muted);
	}

	.condition-type {
		font-weight: 600;
	}

	.condition-message {
		color: var(--text-muted);
		max-width: 200px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.deployment-actions {
		display: flex;
		gap: var(--space-xs);
	}

	.delete-btn:hover {
		color: var(--error);
		background: var(--error-muted);
	}

	.spinner-small {
		width: 16px;
		height: 16px;
		border: 2px solid var(--border-default);
		border-top-color: var(--accent);
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	/* Loading state */
	.loading-list {
		display: flex;
		flex-direction: column;
		gap: var(--space-md);
	}

	.skeleton-row {
		background: var(--bg-elevated);
		border: 1px solid var(--border-subtle);
		border-radius: var(--radius-xl);
		padding: var(--space-xl);
		display: flex;
		align-items: center;
		gap: var(--space-lg);
	}

	.skeleton-icon {
		width: 48px;
		height: 48px;
		border-radius: var(--radius-lg);
		flex-shrink: 0;
	}

	.skeleton-content {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: var(--space-sm);
	}

	.skeleton-title {
		height: 20px;
		width: 40%;
	}

	.skeleton-text {
		height: 14px;
		width: 60%;
	}

	.skeleton-badge {
		width: 80px;
		height: 24px;
		border-radius: var(--radius-xl);
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

	.empty-illustration {
		color: var(--lavender-700);
		margin-bottom: var(--space-xl);
		opacity: 0.5;
	}

	.error-icon {
		color: var(--error);
		margin-bottom: var(--space-lg);
	}

	.empty-state h3,
	.error-state h3 {
		font-size: 1.25rem;
		color: var(--text-primary);
		margin-bottom: var(--space-sm);
	}

	.empty-state p,
	.error-state p {
		color: var(--text-muted);
		margin-bottom: var(--space-xl);
	}

	@media (max-width: 1200px) {
		.deployment-card {
			grid-template-columns: 1fr auto;
		}

		.deployment-details {
			grid-column: 1 / -1;
			grid-template-columns: repeat(4, 1fr);
		}

		.conditions {
			grid-column: 1 / -1;
		}
	}

	@media (max-width: 768px) {
		.page-header {
			flex-direction: column;
			align-items: stretch;
		}

		.page-title {
			font-size: 2rem;
		}

		.deployment-card {
			grid-template-columns: 1fr;
		}

		.deployment-details {
			grid-template-columns: repeat(2, 1fr);
		}

		.deployment-actions {
			justify-content: flex-end;
		}
	}
</style>

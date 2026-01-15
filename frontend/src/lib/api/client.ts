// API Client for App Store Backend

const API_BASE = 'http://localhost:8080/api/v1';

export interface App {
	name: string;
	displayName: string;
	description: string;
	icon: string;
	category: string;
	chartPath: string;
	tags: string[];
}

export interface Condition {
	type: string;
	status: string;
	reason: string;
	message: string;
	lastTransitionTime: string;
}

export interface Deployment {
	name: string;
	namespace: string;
	appName: string;
	teamId: string;
	requestedBy: string;
	phase: string;
	helmReleaseName: string;
	helmReleaseRevision: number;
	deployedChartVersion: string;
	message: string;
	conditions: Condition[];
	createdAt: string;
	lastReconcileTime: string;
}

export interface CreateDeploymentRequest {
	appName: string;
	namespace: string;
	releaseName?: string;
	version?: string;
	values?: Record<string, unknown>;
}

export interface ApiError {
	error: string;
}

class ApiClient {
	private baseUrl: string;

	constructor(baseUrl: string = API_BASE) {
		this.baseUrl = baseUrl;
	}

	private async request<T>(endpoint: string, options?: RequestInit): Promise<T> {
		const url = `${this.baseUrl}${endpoint}`;
		const response = await fetch(url, {
			...options,
			headers: {
				'Content-Type': 'application/json',
				...options?.headers,
			},
		});

		if (!response.ok) {
			const error: ApiError = await response.json().catch(() => ({ error: 'Unknown error' }));
			throw new Error(error.error || `HTTP ${response.status}`);
		}

		return response.json();
	}

	// Catalog endpoints
	async getCatalog(): Promise<{ apps: App[] }> {
		return this.request('/catalog');
	}

	async getApp(appName: string): Promise<App> {
		return this.request(`/catalog/${appName}`);
	}

	// Deployment endpoints
	async getDeployments(namespace?: string): Promise<{ deployments: Deployment[] }> {
		const params = namespace ? `?namespace=${encodeURIComponent(namespace)}` : '';
		return this.request(`/deployments${params}`);
	}

	async getDeployment(name: string, namespace?: string): Promise<Deployment> {
		const params = namespace ? `?namespace=${encodeURIComponent(namespace)}` : '';
		return this.request(`/deployments/${name}${params}`);
	}

	async createDeployment(request: CreateDeploymentRequest): Promise<{ requestId: string; message: string }> {
		return this.request('/deployments', {
			method: 'POST',
			body: JSON.stringify(request),
		});
	}

	async deleteDeployment(name: string, namespace?: string): Promise<{ requestId: string; message: string }> {
		const params = namespace ? `?namespace=${encodeURIComponent(namespace)}` : '';
		return this.request(`/deployments/${name}${params}`, {
			method: 'DELETE',
		});
	}
}

export const api = new ApiClient();
export default api;

import createClient from 'openapi-fetch';
import type { components, paths } from './api.gen';

export type Schemas = components['schemas'];
export type Me = Schemas['Me'];

export const api = createClient<paths>({ baseUrl: '' });

/** ApiError carries the problem detail and, for rejected input, the message of every invalid field. */
export class ApiError extends Error {
	constructor(
		public status: number,
		message: string,
		public errors: string[] = []
	) {
		super(message);
	}
}

type Result<T> = { data?: T; error?: unknown; response: Response };

/** Awaits an openapi-fetch call and returns its data; throws ApiError otherwise. */
export async function call<T>(p: Promise<Result<T>>): Promise<T> {
	const { data, error, response } = await p;
	if (response.ok) return data as T;
	const problem = (error ?? {}) as { detail?: string; errors?: { message?: string }[] };
	const errors = (problem.errors ?? []).map((e) => e.message ?? '').filter(Boolean);
	throw new ApiError(response.status, problem.detail || response.statusText, errors);
}

/** The config query: version and the features the login pages offer. */
export const configQuery = () => ({ queryKey: ['config'], queryFn: () => call(api.GET('/api/config')), staleTime: Infinity });

/** The logged-in user, or null for guests. */
export const meQuery = () => ({
	queryKey: ['me'],
	queryFn: async () => {
		const { data, response } = await api.GET('/api/me');
		if (response.status === 401) return null;
		if (!response.ok) throw new ApiError(response.status, response.statusText);
		return data ?? null;
	}
});

/** Returns next only if it is a local path, otherwise /, so the login cannot redirect off site. */
export function safeNext(next: string | null): string {
	// "//host" and "/\host" (browsers turn \ into /) would leave the site
	if (!next || !next.startsWith('/') || next.startsWith('//') || next.startsWith('/\\')) return '/';
	return next;
}

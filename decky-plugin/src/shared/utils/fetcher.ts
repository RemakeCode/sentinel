import { callable } from '@decky/api';

export const BASE_URL = 'http://localhost:48211/decky-backend';

export const ASSET_URL = BASE_URL.replace('/decky-backend', '');

export const NOTIFICATION_SSE_URL = `${BASE_URL}/notifications`;

const AUTH_QUERY_PARAM = 'decky_auth_token';

const getDeckyAuthTokenCallable = callable<[], string>('get_decky_auth_token');

let authTokenPromise: Promise<string> | null = null;

export function getDeckyAuthToken(): Promise<string> {
  if (!authTokenPromise) {
    authTokenPromise = getDeckyAuthTokenCallable().then((token) => {
      if (!token) {
        authTokenPromise = null;
        throw new Error('Decky auth token is not available');
      }
      return token;
    });
  }
  return authTokenPromise;
}

export async function getNotificationSSEUrl(): Promise<string> {
  const url = new URL(NOTIFICATION_SSE_URL);
  url.searchParams.set(AUTH_QUERY_PARAM, await getDeckyAuthToken());
  return url.toString();
}

export class Fetcher {
  private baseUrl: string;

  constructor(baseUrl?: string) {
    this.baseUrl = baseUrl ? baseUrl : '';
  }

  async get<Type>(url: string): Promise<Type> {
    return fetch(this.url(url), {
      method: 'GET',
      headers: await Fetcher.headers()
    }).then(Fetcher.processResponse);
  }

  async post<Type>(url: string, body: any): Promise<Type> {
    return fetch(this.url(url), {
      method: 'POST',
      headers: await Fetcher.headers(),
      body: JSON.stringify(body)
    }).then(Fetcher.processResponse);
  }

  async delete(url: string): Promise<Response> {
    return fetch(this.url(url), {
      method: 'DELETE',
      headers: await Fetcher.headers(false)
    }).then(Fetcher.processResponse);
  }

  async put<Type>(url: string, body: any): Promise<Type> {
    return fetch(this.url(url), {
      method: 'PUT',
      headers: await Fetcher.headers(),
      body: JSON.stringify(body)
    }).then(Fetcher.processResponse);
  }

  async patch<Type>(url: string, body?: any): Promise<Type> {
    const options: RequestInit = {
      method: 'PATCH',
      headers: await Fetcher.headers()
    };
    if (body !== undefined) {
      options.body = JSON.stringify(body);
    }
    return fetch(this.url(url), options).then(Fetcher.processResponse);
  }

  private url(url: string): string {
    if (url.startsWith('http://') || url.startsWith('https://')) {
      return url;
    }
    return `${this.baseUrl}${url}`;
  }

  private static async headers(includeContentType = true): Promise<Record<string, string>> {
    const headers: Record<string, string> = {
      Authorization: `Bearer ${await getDeckyAuthToken()}`
    };
    if (includeContentType) {
      headers['Content-Type'] = 'application/json';
    }
    return headers;
  }

  private static async processResponse(response: Response) {
    if (response.ok) {
      return await response.json();
    } else {
      const error = await response.json();
      return Promise.reject(error?.message || 'An unknown error occurred');
    }
  }
}

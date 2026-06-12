export const BASE_URL = 'http://localhost:48211/decky-backend';

export const ASSET_URL = BASE_URL.replace('/decky-backend', '');

export const NOTIFICATION_SSE_URL = `${BASE_URL}/notifications`;

export class Fetcher {
  private baseUrl: string;

  constructor(baseUrl?: string) {
    this.baseUrl = baseUrl ? baseUrl : '';
  }

  async get<Type>(url: string): Promise<Type> {
    return fetch(this.url(url), {
      method: 'GET',
      headers: Fetcher.headers()
    }).then(Fetcher.processResponse);
  }

  async post<Type>(url: string, body: any): Promise<Type> {
    return fetch(this.url(url), {
      method: 'POST',
      headers: Fetcher.headers(),
      body: JSON.stringify(body)
    }).then(Fetcher.processResponse);
  }

  async delete(url: string): Promise<Response> {
    return fetch(this.url(url), {
      method: 'DELETE',
      headers: Fetcher.headers(false)
    }).then(Fetcher.processResponse);
  }

  async put<Type>(url: string, body: any): Promise<Type> {
    return fetch(this.url(url), {
      method: 'PUT',
      headers: Fetcher.headers(),
      body: JSON.stringify(body)
    }).then(Fetcher.processResponse);
  }

  async patch<Type>(url: string, body?: any): Promise<Type> {
    const options: RequestInit = {
      method: 'PATCH',
      headers: Fetcher.headers()
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

  private static headers(includeContentType = true): Record<string, string> {
    const headers: Record<string, string> = {};
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

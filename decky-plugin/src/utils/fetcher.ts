const BASE_URL = 'http://localhost:40000/api';
export const NOTIFICATION_SSE_URL = `${BASE_URL}/notifications`;

export class Fetcher {
  async get<Type>(url: string): Promise<Type> {
    return fetch(`${BASE_URL}${url}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json'
      }
    }).then(Fetcher.processResponse);
  }

  async post<Type>(url: string, body: any): Promise<Type> {
    return fetch(`${BASE_URL}${url}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(body)
    }).then(Fetcher.processResponse);
  }

  async delete(url: string): Promise<Response> {
    return fetch(`${BASE_URL}${url}`, {
      method: 'DELETE',
      headers: {}
    }).then(Fetcher.processResponse);
  }

  async put<Type>(url: string, body: any): Promise<Type> {
    return fetch(url, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(body)
    }).then(Fetcher.processResponse);
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

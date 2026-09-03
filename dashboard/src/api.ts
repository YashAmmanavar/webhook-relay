const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";
// Matches the API's default dev key (cmd/api/main.go) so this works out of
// the box against a freshly cloned server with no env vars set. Never reuse
// this beyond localhost.
const API_KEY = import.meta.env.VITE_API_KEY ?? "sk_dev_local_only_insecure_key";

export type EventStatus = "PENDING" | "PROCESSING" | "RETRYING" | "DELIVERED" | "FAILED";

export interface Event {
  id: string;
  endpoint_id: string;
  event_type: string;
  payload: unknown;
  status: EventStatus;
  attempt_count: number;
  next_retry_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface Attempt {
  id: number;
  event_id: string;
  attempt_number: number;
  status_code: number | null;
  response_body: string | null;
  error: string | null;
  duration_ms: number | null;
  created_at: string;
}

export interface EventDetail extends Event {
  attempts: Attempt[];
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers: {
      ...init?.headers,
      Authorization: `Bearer ${API_KEY}`,
    },
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? `request failed with status ${res.status}`);
  }
  return res.json() as Promise<T>;
}

export function listEvents(status?: EventStatus): Promise<Event[]> {
  const query = status ? `?status=${status}` : "";
  return request<Event[]>(`/api/v1/events${query}`);
}

export function getEvent(id: string): Promise<EventDetail> {
  return request<EventDetail>(`/api/v1/events/${id}`);
}

export function retryEvent(id: string): Promise<{ event_id: string; status: EventStatus }> {
  return request(`/api/v1/events/${id}/retry`, { method: "POST" });
}

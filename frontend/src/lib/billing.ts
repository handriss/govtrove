export const PRO_FEATURES_FREE_FOR_ALL = true;

export const DONATION_URL = 'https://donate.stripe.com/3cI3cv2NM5XherJgFO2VG00';

const API_BASE = import.meta.env.VITE_API_URL || '/api';

export async function trackDonationClick(source: string, token?: string): Promise<void> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (token) headers['Authorization'] = `Bearer ${token}`;
  try {
    await fetch(`${API_BASE}/track-donation-click`, {
      method: 'POST',
      headers,
      body: JSON.stringify({ source }),
      keepalive: true,
    });
  } catch {
    // fire-and-forget
  }
}

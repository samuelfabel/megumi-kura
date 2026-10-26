import { ApiError, createPublicPromise, type PromiseInput } from "./api";

const QUEUE_KEY = "mk_promise_queue_v1";

type QueuedPromise = PromiseInput & { queued_at: string };

function stripCaptcha(input: PromiseInput): PromiseInput {
  return {
    food_id: input.food_id,
    person_name: input.person_name,
    quantity: input.quantity,
    promise_date: input.promise_date,
  };
}

function readQueue(): QueuedPromise[] {
  try {
    const raw = localStorage.getItem(QUEUE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as QueuedPromise[];
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function writeQueue(items: QueuedPromise[]) {
  localStorage.setItem(QUEUE_KEY, JSON.stringify(items));
}

export type SubmitPromiseResult =
  | { status: "sent"; id: number }
  | { status: "queued" };

function isNetworkFailure(err: unknown): boolean {
  if (err instanceof ApiError) {
    return err.status >= 500;
  }
  return err instanceof TypeError || (err instanceof Error && /failed to fetch|network/i.test(err.message));
}

/** Submit now (with captcha), or queue locally when offline / network fails. */
export async function submitPromiseOfflineFirst(
  input: PromiseInput,
): Promise<SubmitPromiseResult> {
  if (!navigator.onLine) {
    enqueue(stripCaptcha(input));
    return { status: "queued" };
  }
  try {
    const res = await createPublicPromise(input);
    return { status: "sent", id: res.id };
  } catch (err) {
    if (isNetworkFailure(err)) {
      enqueue(stripCaptcha(input));
      return { status: "queued" };
    }
    throw err;
  }
}

function enqueue(input: PromiseInput) {
  const queue = readQueue();
  queue.push({ ...input, queued_at: new Date().toISOString() });
  writeQueue(queue);
}

export function pendingPromiseCount(): number {
  return readQueue().length;
}

/**
 * Send the oldest queued promise using a fresh captcha.
 * Captcha is required by the API, so silent auto-flush is not used.
 */
export async function flushNextQueuedPromise(captcha: {
  captcha_token: string;
  captcha_answer: string;
}): Promise<boolean> {
  const queue = readQueue();
  if (queue.length === 0) return false;
  const [first, ...rest] = queue;
  const { queued_at: _, captcha_token: _t, captcha_answer: _a, ...payload } = first;
  await createPublicPromise({
    ...payload,
    captcha_token: captcha.captcha_token,
    captcha_answer: captcha.captcha_answer,
  });
  writeQueue(rest);
  return true;
}

/** Notify when connectivity returns so the UI can prompt for captcha flush. */
export function watchOnline(onOnline?: () => void) {
  const handler = () => onOnline?.();
  window.addEventListener("online", handler);
  return () => window.removeEventListener("online", handler);
}

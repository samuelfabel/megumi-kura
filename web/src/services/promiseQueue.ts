import { createPublicPromise, type PromiseInput } from "./api";

const QUEUE_KEY = "mk_promise_queue_v1";

type QueuedPromise = PromiseInput & { queued_at: string };

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

/** Submit now, or queue locally when offline / network fails. */
export async function submitPromiseOfflineFirst(
  input: PromiseInput,
): Promise<SubmitPromiseResult> {
  if (!navigator.onLine) {
    enqueue(input);
    return { status: "queued" };
  }
  try {
    const res = await createPublicPromise(input);
    return { status: "sent", id: res.id };
  } catch {
    enqueue(input);
    return { status: "queued" };
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

/** Flush queued promises when connectivity returns. */
export async function flushPromiseQueue(): Promise<number> {
  const queue = readQueue();
  if (queue.length === 0) return 0;

  const remaining: QueuedPromise[] = [];
  let sent = 0;
  for (const item of queue) {
    try {
      const { queued_at: _, ...payload } = item;
      await createPublicPromise(payload);
      sent += 1;
    } catch {
      remaining.push(item);
    }
  }
  writeQueue(remaining);
  return sent;
}

export function watchOnlineFlush(onFlushed?: (sent: number) => void) {
  const handler = () => {
    void flushPromiseQueue().then((sent) => {
      if (sent > 0) onFlushed?.(sent);
    });
  };
  window.addEventListener("online", handler);
  // Attempt once on load as well.
  handler();
  return () => window.removeEventListener("online", handler);
}

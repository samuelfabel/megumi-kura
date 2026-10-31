import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError, createPublicPromise, type PromiseInput } from "./api";
import {
  flushNextQueuedPromise,
  pendingPromiseCount,
  submitPromiseOfflineFirst,
} from "./promiseQueue";

vi.mock("./api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./api")>();
  return { ...actual, createPublicPromise: vi.fn() };
});

const create = vi.mocked(createPublicPromise);

const input: PromiseInput = {
  food_id: 1,
  person_name: "Ana",
  quantity: "2",
  promise_date: "2026-11-01",
  captcha_token: "tok",
  captcha_answer: "7",
};

function setOnline(value: boolean) {
  Object.defineProperty(navigator, "onLine", { configurable: true, get: () => value });
}

function storedQueue(): Array<Record<string, unknown>> {
  return JSON.parse(localStorage.getItem("mk_promise_queue_v1") ?? "[]");
}

beforeEach(() => {
  localStorage.clear();
  create.mockReset();
  setOnline(true);
});

afterEach(() => {
  setOnline(true);
});

describe("submitPromiseOfflineFirst", () => {
  it("sends immediately when online", async () => {
    create.mockResolvedValue({ id: 42, food_id: 1, quantity: "2", promise_date: "2026-11-01" });

    await expect(submitPromiseOfflineFirst(input)).resolves.toEqual({ status: "sent", id: 42 });
    expect(create).toHaveBeenCalledWith(input);
    expect(pendingPromiseCount()).toBe(0);
  });

  it("queues without the captcha when offline", async () => {
    setOnline(false);

    await expect(submitPromiseOfflineFirst(input)).resolves.toEqual({ status: "queued" });
    expect(create).not.toHaveBeenCalled();
    const [queued] = storedQueue();
    expect(queued).toMatchObject({ food_id: 1, person_name: "Ana", quantity: "2" });
    expect(queued).not.toHaveProperty("captcha_token");
    expect(queued).not.toHaveProperty("captcha_answer");
  });

  it("queues on server errors and network failures", async () => {
    create.mockRejectedValueOnce(new ApiError(503, "unavailable"));
    create.mockRejectedValueOnce(new TypeError("Failed to fetch"));

    await expect(submitPromiseOfflineFirst(input)).resolves.toEqual({ status: "queued" });
    await expect(submitPromiseOfflineFirst(input)).resolves.toEqual({ status: "queued" });
    expect(pendingPromiseCount()).toBe(2);
  });

  it("rethrows client errors instead of queueing", async () => {
    create.mockRejectedValue(new ApiError(400, "invalid captcha"));

    await expect(submitPromiseOfflineFirst(input)).rejects.toThrow("invalid captcha");
    expect(pendingPromiseCount()).toBe(0);
  });
});

describe("flushNextQueuedPromise", () => {
  it("returns false when the queue is empty", async () => {
    await expect(flushNextQueuedPromise({ captcha_token: "t", captcha_answer: "1" })).resolves.toBe(false);
    expect(create).not.toHaveBeenCalled();
  });

  it("sends the oldest promise with a fresh captcha and removes it", async () => {
    setOnline(false);
    await submitPromiseOfflineFirst({ ...input, person_name: "Primeira" });
    await submitPromiseOfflineFirst({ ...input, person_name: "Segunda" });
    create.mockResolvedValue({ id: 1, food_id: 1, quantity: "2", promise_date: "2026-11-01" });

    await expect(flushNextQueuedPromise({ captcha_token: "novo", captcha_answer: "9" })).resolves.toBe(true);

    expect(create).toHaveBeenCalledWith({
      food_id: 1,
      person_name: "Primeira",
      quantity: "2",
      promise_date: "2026-11-01",
      captcha_token: "novo",
      captcha_answer: "9",
    });
    expect(storedQueue().map((p) => p.person_name)).toEqual(["Segunda"]);
  });

  it("keeps the promise queued when sending fails", async () => {
    setOnline(false);
    await submitPromiseOfflineFirst(input);
    create.mockRejectedValue(new ApiError(400, "invalid captcha"));

    await expect(flushNextQueuedPromise({ captcha_token: "t", captcha_answer: "0" })).rejects.toThrow();
    expect(pendingPromiseCount()).toBe(1);
  });

  it("treats a corrupted queue as empty", () => {
    localStorage.setItem("mk_promise_queue_v1", "{not json");
    expect(pendingPromiseCount()).toBe(0);
  });
});

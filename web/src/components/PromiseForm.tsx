import { useCallback, useEffect, useState, type FormEvent } from "react";
import { t } from "../i18n";
import { ApiError, fetchCaptcha, type StockItem } from "../services/api";
import {
  flushNextQueuedPromise,
  pendingPromiseCount,
  submitPromiseOfflineFirst,
  watchOnline,
} from "../services/promiseQueue";

type Props = {
  foods: StockItem[];
  onSubmitted: () => void;
};

function todayISO() {
  const d = new Date();
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

export function PromiseForm({ foods, onSubmitted }: Props) {
  const [foodId, setFoodId] = useState(0);
  const [person, setPerson] = useState("");
  const [qty, setQty] = useState("");
  const [date, setDate] = useState(todayISO());
  const [busy, setBusy] = useState(false);
  const [feedback, setFeedback] = useState<"sent" | "queued" | "flushed" | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(0);
  const [captchaToken, setCaptchaToken] = useState("");
  const [captchaQuestion, setCaptchaQuestion] = useState("");
  const [captchaAnswer, setCaptchaAnswer] = useState("");

  const refreshCaptcha = useCallback(async () => {
    try {
      const ch = await fetchCaptcha();
      setCaptchaToken(ch.captcha_token);
      setCaptchaQuestion(ch.question);
      setCaptchaAnswer("");
    } catch {
      setCaptchaToken("");
      setCaptchaQuestion("");
      setError(t("promise.error_captcha_load"));
    }
  }, []);

  useEffect(() => {
    if (foods[0] && foodId === 0) setFoodId(foods[0].food_id);
  }, [foods, foodId]);

  useEffect(() => {
    setPending(pendingPromiseCount());
    void refreshCaptcha();
    return watchOnline(() => setPending(pendingPromiseCount()));
  }, [refreshCaptcha]);

  const selected = foods.find((f) => f.food_id === foodId);

  function mapCaptchaError(err: unknown): string {
    if (err instanceof ApiError) {
      if (err.message.includes("captcha")) {
        if (err.message.includes("expired")) return t("promise.error_captcha_expired");
        if (err.message.includes("used")) return t("promise.error_captcha_used");
        return t("promise.error_captcha");
      }
      return err.message || t("promise.error_generic");
    }
    return t("promise.error_generic");
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setFeedback(null);
    if (!foodId || !person.trim() || !qty.trim() || !date) {
      setError(t("promise.error_required"));
      return;
    }
    if (navigator.onLine && (!captchaToken || !captchaAnswer.trim())) {
      setError(t("promise.error_captcha_required"));
      return;
    }
    setBusy(true);
    try {
      const result = await submitPromiseOfflineFirst({
        food_id: foodId,
        person_name: person.trim(),
        quantity: qty.trim(),
        promise_date: date,
        captcha_token: captchaToken,
        captcha_answer: captchaAnswer.trim(),
      });
      setFeedback(result.status);
      setPending(pendingPromiseCount());
      setPerson("");
      setQty("");
      await refreshCaptcha();
      if (result.status === "sent") onSubmitted();
    } catch (err) {
      setError(mapCaptchaError(err));
      await refreshCaptcha();
    } finally {
      setBusy(false);
    }
  }

  async function onFlushPending() {
    setError(null);
    setFeedback(null);
    if (!captchaToken || !captchaAnswer.trim()) {
      setError(t("promise.error_captcha_required"));
      return;
    }
    setBusy(true);
    try {
      const sent = await flushNextQueuedPromise({
        captcha_token: captchaToken,
        captcha_answer: captchaAnswer.trim(),
      });
      setPending(pendingPromiseCount());
      await refreshCaptcha();
      if (sent) {
        setFeedback("flushed");
        onSubmitted();
      }
    } catch (err) {
      setError(mapCaptchaError(err));
      await refreshCaptcha();
    } finally {
      setBusy(false);
    }
  }

  return (
    <section id="doar" className="section promise-section reveal" style={{ animationDelay: "200ms" }}>
      <h2>{t("promise.title")}</h2>
      <p className="section-lead">{t("promise.lead")}</p>

      <form className="promise-form" onSubmit={(e) => void onSubmit(e)}>
        <label>
          <span>{t("promise.person")}</span>
          <input
            value={person}
            onChange={(e) => setPerson(e.target.value)}
            autoComplete="name"
            name="person_name"
            required
          />
        </label>

        <label>
          <span>{t("promise.food")}</span>
          <select
            value={foodId}
            onChange={(e) => setFoodId(Number(e.target.value))}
            required
          >
            {foods.map((f) => (
              <option key={f.food_id} value={f.food_id}>
                {f.name} ({f.unit})
              </option>
            ))}
          </select>
        </label>

        <div className="promise-row">
          <label>
            <span>
              {t("promise.quantity")}
              {selected ? ` (${selected.unit})` : ""}
            </span>
            <input
              inputMode="decimal"
              value={qty}
              onChange={(e) => setQty(e.target.value)}
              placeholder="0"
              required
            />
          </label>
          <label>
            <span>{t("promise.date")}</span>
            <input
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
              required
            />
          </label>
        </div>

        <div className="captcha-row">
          <label className="captcha-field">
            <span>{t("promise.captcha")}</span>
            <div className="captcha-input-row">
              <span className="captcha-question" aria-live="polite">
                {captchaQuestion ? `${captchaQuestion} =` : "…"}
              </span>
              <input
                inputMode="numeric"
                value={captchaAnswer}
                onChange={(e) => setCaptchaAnswer(e.target.value)}
                name="captcha_answer"
                autoComplete="off"
                required={typeof navigator !== "undefined" ? navigator.onLine : true}
                aria-label={t("promise.captcha")}
              />
              <button
                type="button"
                className="btn-secondary captcha-refresh"
                onClick={() => void refreshCaptcha()}
                disabled={busy}
                aria-label={t("promise.captcha_refresh")}
              >
                {t("promise.captcha_refresh")}
              </button>
            </div>
          </label>
        </div>

        <button type="submit" className="btn-primary" disabled={busy || foods.length === 0}>
          {busy ? t("promise.submitting") : t("promise.submit")}
        </button>

        {feedback === "sent" && <p className="ok-text">{t("promise.success")}</p>}
        {feedback === "queued" && <p className="ok-text">{t("promise.queued")}</p>}
        {feedback === "flushed" && <p className="ok-text">{t("promise.flushed")}</p>}
        {error && <p className="error-text">{error}</p>}
        {pending > 0 && (
          <div className="pending-box">
            <p className="muted pending-note">
              {t("promise.pending").replace("{count}", String(pending))}
            </p>
            <button
              type="button"
              className="btn-secondary"
              disabled={busy || !navigator.onLine}
              onClick={() => void onFlushPending()}
            >
              {t("promise.flush_one")}
            </button>
          </div>
        )}
        <p className="privacy-note">{t("promise.privacy")}</p>
      </form>
    </section>
  );
}

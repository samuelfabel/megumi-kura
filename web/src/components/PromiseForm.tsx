import { useEffect, useState, type FormEvent } from "react";
import { CaptchaField, type CaptchaValue } from "./CaptchaField";
import { t } from "../i18n";
import {
  ApiError,
  createPublicPromiseBatch,
  type Kit,
  type StockItem,
} from "../services/api";
import {
  flushNextQueuedPromise,
  pendingPromiseCount,
  submitPromiseOfflineFirst,
  watchOnline,
} from "../services/promiseQueue";

type Line = { food_id: number; quantity: string };

type Props = {
  foods: StockItem[];
  kits?: Kit[];
  onSubmitted: () => void;
};

function todayISO() {
  const d = new Date();
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

export function PromiseForm({ foods, kits = [], onSubmitted }: Props) {
  const [person, setPerson] = useState("");
  const [date, setDate] = useState(todayISO());
  const [mode, setMode] = useState<"items" | "kit">("items");
  const [kitId, setKitId] = useState(0);
  const [lines, setLines] = useState<Line[]>([{ food_id: 0, quantity: "" }]);
  const [busy, setBusy] = useState(false);
  const [feedback, setFeedback] = useState<"sent" | "queued" | "flushed" | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(0);
  const [captcha, setCaptcha] = useState<CaptchaValue | null>(null);
  const [captchaReset, setCaptchaReset] = useState(0);

  useEffect(() => {
    if (foods[0] && lines[0]?.food_id === 0) {
      setLines([{ food_id: foods[0].food_id, quantity: "" }]);
    }
  }, [foods, lines]);

  useEffect(() => {
    if (kits[0] && kitId === 0) setKitId(kits[0].id);
  }, [kits, kitId]);

  useEffect(() => {
    setPending(pendingPromiseCount());
    return watchOnline(() => setPending(pendingPromiseCount()));
  }, []);

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

  function captchaReady(): boolean {
    if (!captcha?.captcha_token) return false;
    if (captcha.provider === "math") return Boolean(captcha.captcha_answer.trim());
    return true;
  }

  function bumpCaptcha() {
    setCaptcha(null);
    setCaptchaReset((n) => n + 1);
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setFeedback(null);
    if (!person.trim() || !date) {
      setError(t("promise.error_required"));
      return;
    }
    if (navigator.onLine && !captchaReady()) {
      setError(t("promise.error_captcha_required"));
      return;
    }

    const items =
      mode === "kit"
        ? []
        : lines
            .filter((l) => l.food_id && l.quantity.trim())
            .map((l) => ({
              food_id: l.food_id,
              quantity: l.quantity.trim(),
            }));
    if (mode === "items" && items.length === 0) {
      setError(t("promise.error_required"));
      return;
    }
    if (mode === "kit" && !kitId) {
      setError(t("promise.error_kit"));
      return;
    }

    setBusy(true);
    try {
      if (!navigator.onLine) {
        if (items[0]) {
          await submitPromiseOfflineFirst({
            food_id: items[0].food_id,
            person_name: person.trim(),
            quantity: items[0].quantity,
            promise_date: date,
          });
          setFeedback("queued");
        } else {
          setError(t("promise.error_offline_kit"));
        }
      } else {
        await createPublicPromiseBatch({
          person_name: person.trim(),
          promise_date: date,
          kit_id: mode === "kit" ? kitId : null,
          items,
          captcha_token: captcha!.captcha_token,
          captcha_answer: captcha!.captcha_answer || "",
        });
        setFeedback("sent");
        onSubmitted();
      }
      setPending(pendingPromiseCount());
      setPerson("");
      setLines([{ food_id: foods[0]?.food_id ?? 0, quantity: "" }]);
      bumpCaptcha();
    } catch (err) {
      setError(mapCaptchaError(err));
      bumpCaptcha();
    } finally {
      setBusy(false);
    }
  }

  async function onFlushPending() {
    setError(null);
    setFeedback(null);
    if (!captchaReady()) {
      setError(t("promise.error_captcha_required"));
      return;
    }
    setBusy(true);
    try {
      const sent = await flushNextQueuedPromise({
        captcha_token: captcha!.captcha_token,
        captcha_answer: captcha!.captcha_answer || "",
      });
      setPending(pendingPromiseCount());
      bumpCaptcha();
      if (sent) {
        setFeedback("flushed");
        onSubmitted();
      }
    } catch (err) {
      setError(mapCaptchaError(err));
      bumpCaptcha();
    } finally {
      setBusy(false);
    }
  }

  const selectedKit = kits.find((k) => k.id === kitId);

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
          <span>{t("promise.date")}</span>
          <input type="date" value={date} onChange={(e) => setDate(e.target.value)} required />
        </label>

        <div className="mode-row">
          <button
            type="button"
            className={mode === "items" ? "btn-secondary active" : "btn-secondary"}
            onClick={() => setMode("items")}
          >
            {t("promise.mode_items")}
          </button>
          <button
            type="button"
            className={mode === "kit" ? "btn-secondary active" : "btn-secondary"}
            onClick={() => setMode("kit")}
            disabled={kits.length === 0}
          >
            {t("promise.mode_kit")}
          </button>
        </div>

        {mode === "kit" ? (
          <label>
            <span>{t("promise.kit")}</span>
            <select value={kitId} onChange={(e) => setKitId(Number(e.target.value))}>
              {kits.map((k) => (
                <option key={k.id} value={k.id}>
                  {k.name}
                </option>
              ))}
            </select>
            {selectedKit?.items && selectedKit.items.length > 0 && (
              <ul className="kit-preview">
                {selectedKit.items.map((it) => (
                  <li key={it.food_id}>
                    {it.name}: {it.quantity} {it.unit}
                  </li>
                ))}
              </ul>
            )}
          </label>
        ) : (
          <div className="lines">
            {lines.map((line, idx) => (
              <div className="promise-row" key={idx}>
                <label>
                  <span>{t("promise.food")}</span>
                  <select
                    value={line.food_id}
                    onChange={(e) => {
                      const next = [...lines];
                      next[idx] = { ...next[idx], food_id: Number(e.target.value) };
                      setLines(next);
                    }}
                  >
                    {foods.map((f) => (
                      <option key={f.food_id} value={f.food_id}>
                        {f.name} ({f.unit})
                      </option>
                    ))}
                  </select>
                </label>
                <label>
                  <span>{t("promise.quantity")}</span>
                  <input
                    inputMode="decimal"
                    value={line.quantity}
                    onChange={(e) => {
                      const next = [...lines];
                      next[idx] = { ...next[idx], quantity: e.target.value };
                      setLines(next);
                    }}
                    required
                  />
                </label>
              </div>
            ))}
            <button
              type="button"
              className="btn-secondary"
              onClick={() =>
                setLines([...lines, { food_id: foods[0]?.food_id ?? 0, quantity: "" }])
              }
            >
              {t("promise.add_line")}
            </button>
          </div>
        )}

        <CaptchaField onChange={setCaptcha} resetKey={captchaReset} />

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

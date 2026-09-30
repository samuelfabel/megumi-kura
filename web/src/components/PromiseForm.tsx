import { useEffect, useState, type FormEvent } from "react";
import { t } from "../i18n";
import type { StockItem } from "../services/api";
import {
  pendingPromiseCount,
  submitPromiseOfflineFirst,
  watchOnlineFlush,
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
  const [feedback, setFeedback] = useState<"sent" | "queued" | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(0);

  useEffect(() => {
    if (foods[0] && foodId === 0) setFoodId(foods[0].food_id);
  }, [foods, foodId]);

  useEffect(() => {
    setPending(pendingPromiseCount());
    return watchOnlineFlush((sent) => {
      if (sent > 0) {
        setPending(pendingPromiseCount());
        onSubmitted();
      }
    });
  }, [onSubmitted]);

  const selected = foods.find((f) => f.food_id === foodId);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setFeedback(null);
    if (!foodId || !person.trim() || !qty.trim() || !date) {
      setError(t("promise.error_required"));
      return;
    }
    setBusy(true);
    try {
      const result = await submitPromiseOfflineFirst({
        food_id: foodId,
        person_name: person.trim(),
        quantity: qty.trim(),
        promise_date: date,
      });
      setFeedback(result.status);
      setPending(pendingPromiseCount());
      setPerson("");
      setQty("");
      if (result.status === "sent") onSubmitted();
    } catch {
      setError(t("promise.error_generic"));
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

        <button type="submit" className="btn-primary" disabled={busy || foods.length === 0}>
          {busy ? t("promise.submitting") : t("promise.submit")}
        </button>

        {feedback === "sent" && <p className="ok-text">{t("promise.success")}</p>}
        {feedback === "queued" && <p className="ok-text">{t("promise.queued")}</p>}
        {error && <p className="error-text">{error}</p>}
        {pending > 0 && (
          <p className="muted pending-note">
            {t("promise.pending").replace("{count}", String(pending))}
          </p>
        )}
        <p className="privacy-note">{t("promise.privacy")}</p>
      </form>
    </section>
  );
}

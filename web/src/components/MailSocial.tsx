import { useEffect, useState, type FormEvent } from "react";
import { t } from "../i18n";
import { subscribeMail } from "../services/api";

export function MailSubscribe() {
  const [email, setEmail] = useState("");
  const [busy, setBusy] = useState(false);
  const [ok, setOk] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    setOk(false);
    try {
      await subscribeMail(email.trim());
      setOk(true);
      setEmail("");
    } catch {
      setError(t("mail.error"));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="section reveal" style={{ animationDelay: "360ms" }}>
      <h2>{t("mail.title")}</h2>
      <p className="section-lead">{t("mail.lead")}</p>
      <form className="promise-form" onSubmit={(e) => void onSubmit(e)}>
        <label>
          <span>{t("mail.email")}</span>
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            autoComplete="email"
          />
        </label>
        <button type="submit" className="btn-primary" disabled={busy}>
          {busy ? t("mail.submitting") : t("mail.submit")}
        </button>
        {ok && <p className="ok-text">{t("mail.success")}</p>}
        {error && <p className="error-text">{error}</p>}
      </form>
    </section>
  );
}

export function SocialTeaser({ onOpen }: { onOpen: () => void }) {
  useEffect(() => {}, []);
  return (
    <section className="section reveal" style={{ animationDelay: "400ms" }}>
      <h2>{t("social.title")}</h2>
      <p className="section-lead">{t("social.lead")}</p>
      <button type="button" className="btn-secondary" onClick={onOpen}>
        {t("social.open")}
      </button>
    </section>
  );
}

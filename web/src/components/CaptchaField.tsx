import { useEffect, useId, useRef, useState } from "react";
import { t } from "../i18n";
import { fetchCaptcha, type CaptchaChallenge } from "../services/api";

export type CaptchaValue = {
  provider: string;
  captcha_token: string;
  captcha_answer: string;
};

type Props = {
  onChange: (value: CaptchaValue | null) => void;
  resetKey?: number;
};

declare global {
  interface Window {
    grecaptcha?: {
      render: (
        el: HTMLElement,
        opts: { sitekey: string; callback: (token: string) => void; "expired-callback"?: () => void },
      ) => number;
      reset: (id?: number) => void;
    };
    turnstile?: {
      render: (
        el: HTMLElement,
        opts: { sitekey: string; callback: (token: string) => void; "expired-callback"?: () => void },
      ) => string;
      reset: (id?: string) => void;
    };
  }
}

function loadScript(src: string, id: string): Promise<void> {
  const existing = document.getElementById(id);
  if (existing) return Promise.resolve();
  return new Promise((resolve, reject) => {
    const s = document.createElement("script");
    s.id = id;
    s.src = src;
    s.async = true;
    s.onload = () => resolve();
    s.onerror = () => reject(new Error("script load failed"));
    document.head.appendChild(s);
  });
}

export function CaptchaField({ onChange, resetKey = 0 }: Props) {
  const mountId = useId().replace(/:/g, "");
  const widgetRef = useRef<HTMLDivElement | null>(null);
  const widgetIdRef = useRef<number | string | null>(null);
  const [challenge, setChallenge] = useState<CaptchaChallenge | null>(null);
  const [answer, setAnswer] = useState("");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setError(null);
    setAnswer("");
    onChange(null);
    void fetchCaptcha()
      .then(async (ch) => {
        if (cancelled) return;
        setChallenge(ch);
        if (ch.provider === "math") {
          onChange({
            provider: "math",
            captcha_token: ch.captcha_token || "",
            captcha_answer: "",
          });
          return;
        }
        if (!ch.site_key) {
          setError(t("promise.error_captcha_load"));
          return;
        }
        if (ch.provider === "google") {
          await loadScript("https://www.google.com/recaptcha/api.js?render=explicit", "mk-recaptcha");
          if (cancelled || !widgetRef.current || !window.grecaptcha) return;
          widgetRef.current.innerHTML = "";
          widgetIdRef.current = window.grecaptcha.render(widgetRef.current, {
            sitekey: ch.site_key,
            callback: (token: string) =>
              onChange({ provider: "google", captcha_token: token, captcha_answer: "" }),
            "expired-callback": () => onChange(null),
          });
        } else if (ch.provider === "cloudflare") {
          await loadScript(
            "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit",
            "mk-turnstile",
          );
          if (cancelled || !widgetRef.current || !window.turnstile) return;
          widgetRef.current.innerHTML = "";
          widgetIdRef.current = window.turnstile.render(widgetRef.current, {
            sitekey: ch.site_key,
            callback: (token: string) =>
              onChange({ provider: "cloudflare", captcha_token: token, captcha_answer: "" }),
            "expired-callback": () => onChange(null),
          });
        } else {
          setError(t("promise.error_captcha_load"));
        }
      })
      .catch(() => {
        if (!cancelled) setError(t("promise.error_captcha_load"));
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [resetKey]);

  if (error) {
    return <p className="error-text">{error}</p>;
  }

  if (!challenge) {
    return <p className="muted">{t("promise.captcha_loading")}</p>;
  }

  if (challenge.provider === "math") {
    return (
      <div className="captcha-row">
        <label className="captcha-field">
          <span>{t("promise.captcha")}</span>
          <div className="captcha-input-row">
            <span className="captcha-question" aria-live="polite">
              {challenge.question ? `${challenge.question} =` : "…"}
            </span>
            <input
              inputMode="numeric"
              value={answer}
              onChange={(e) => {
                const v = e.target.value;
                setAnswer(v);
                onChange({
                  provider: "math",
                  captcha_token: challenge.captcha_token || "",
                  captcha_answer: v.trim(),
                });
              }}
              name="captcha_answer"
              autoComplete="off"
              required={typeof navigator !== "undefined" ? navigator.onLine : true}
            />
          </div>
        </label>
      </div>
    );
  }

  return (
    <div className="captcha-row">
      <span className="captcha-label">{t("promise.captcha")}</span>
      <div id={`mk-captcha-${mountId}`} ref={widgetRef} className="captcha-widget" />
    </div>
  );
}

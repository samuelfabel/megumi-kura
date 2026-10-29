import { useEffect, useState, type FormEvent } from "react";
import { t } from "../i18n";
import {
  createKit,
  createSocialPost,
  createUser,
  getMailSettings,
  listAdminKits,
  listMailTemplates,
  listSocialAdmin,
  listUsers,
  moderateComment,
  putKitItems,
  putMailSettings,
  sendShortageMail,
  updateMailTemplate,
  updateUser,
  type Food,
  type Kit,
} from "../services/api";

export function KitsPanel({ foods }: { foods: Food[] }) {
  const [kits, setKits] = useState<Kit[]>([]);
  const [code, setCode] = useState("");
  const [name, setName] = useState("");
  const [editId, setEditId] = useState(0);
  const [items, setItems] = useState<Array<{ food_id: number; quantity: string }>>([
    { food_id: foods[0]?.id ?? 0, quantity: "1" },
  ]);
  const [msg, setMsg] = useState<string | null>(null);

  async function reload() {
    const r = await listAdminKits();
    setKits(r.kits ?? []);
  }

  useEffect(() => {
    void reload().catch(() => setMsg(t("admin.error")));
  }, []);

  async function onCreate(e: FormEvent) {
    e.preventDefault();
    try {
      const k = await createKit({ code, name });
      setEditId(k.id);
      setCode("");
      setName("");
      await reload();
      setMsg(t("admin.kit_created"));
    } catch (err) {
      setMsg(err instanceof Error ? err.message : t("admin.error"));
    }
  }

  async function onSaveItems(e: FormEvent) {
    e.preventDefault();
    if (!editId) return;
    try {
      await putKitItems(
        editId,
        items.filter((i) => i.food_id && i.quantity),
      );
      await reload();
      setMsg(t("admin.kit_saved"));
    } catch (err) {
      setMsg(err instanceof Error ? err.message : t("admin.error"));
    }
  }

  return (
    <div className="admin-panel">
      <h2>{t("admin.tab_kits")}</h2>
      <p className="muted">{t("admin.kits_lead")}</p>
      <ul>
        {kits.map((k) => (
          <li key={k.id}>
            <button type="button" className="btn-secondary" onClick={() => setEditId(k.id)}>
              {k.name} ({k.code}) — {k.available ?? 0}
            </button>
          </li>
        ))}
      </ul>
      <form className="admin-form" onSubmit={(e) => void onCreate(e)}>
        <h3>{t("admin.kit_new")}</h3>
        <label>
          <span>code</span>
          <input value={code} onChange={(e) => setCode(e.target.value)} required />
        </label>
        <label>
          <span>{t("admin.name")}</span>
          <input value={name} onChange={(e) => setName(e.target.value)} required />
        </label>
        <button type="submit" className="btn-primary">
          {t("admin.create")}
        </button>
      </form>
      {editId > 0 && (
        <form className="admin-form" onSubmit={(e) => void onSaveItems(e)}>
          <h3>
            {t("admin.kit_edit_items")} #{editId}
          </h3>
          {items.map((it, idx) => (
            <div className="promise-row" key={idx}>
              <select
                value={it.food_id}
                onChange={(e) => {
                  const next = [...items];
                  next[idx] = { ...next[idx], food_id: Number(e.target.value) };
                  setItems(next);
                }}
              >
                {foods.map((f) => (
                  <option key={f.id} value={f.id}>
                    {f.name}
                  </option>
                ))}
              </select>
              <input
                value={it.quantity}
                onChange={(e) => {
                  const next = [...items];
                  next[idx] = { ...next[idx], quantity: e.target.value };
                  setItems(next);
                }}
              />
            </div>
          ))}
          <button
            type="button"
            className="btn-secondary"
            onClick={() => setItems([...items, { food_id: foods[0]?.id ?? 0, quantity: "1" }])}
          >
            +
          </button>
          <button type="submit" className="btn-primary">
            {t("admin.save")}
          </button>
        </form>
      )}
      {msg && <p className="ok-text">{msg}</p>}
    </div>
  );
}

export function MailPanel() {
  const [templates, setTemplates] = useState<
    Array<{ key: string; name: string; subject: string; body_html: string; required_tags: string[] }>
  >([]);
  const [key, setKey] = useState("shortage_alert");
  const [subject, setSubject] = useState("");
  const [body, setBody] = useState("");
  const [smtpHost, setSmtpHost] = useState("");
  const [smtpPort, setSmtpPort] = useState("587");
  const [smtpUser, setSmtpUser] = useState("");
  const [smtpPass, setSmtpPass] = useState("");
  const [smtpFrom, setSmtpFrom] = useState("");
  const [auto, setAuto] = useState(false);
  const [freq, setFreq] = useState("daily");
  const [threshold, setThreshold] = useState("0");
  const [subs, setSubs] = useState(0);
  const [msg, setMsg] = useState<string | null>(null);

  useEffect(() => {
    void (async () => {
      const [tpl, settings] = await Promise.all([listMailTemplates(), getMailSettings()]);
      setTemplates(tpl.templates ?? []);
      const cur = (tpl.templates ?? []).find((x) => x.key === key);
      if (cur) {
        setSubject(cur.subject);
        setBody(cur.body_html);
      }
      setSmtpHost(settings.smtp?.host ?? "");
      setSmtpPort(settings.smtp?.port || "587");
      setSmtpUser(settings.smtp?.user ?? "");
      setSmtpFrom(settings.smtp?.from ?? "");
      setAuto(Boolean(settings.alerts?.auto_enabled ?? settings.alerts?.AutoEnabled));
      setFreq(settings.alerts?.auto_frequency || settings.alerts?.AutoFrequency || "daily");
      setSubs(settings.subscriber_count ?? 0);
    })().catch(() => setMsg(t("admin.error")));
  }, [key]);

  async function saveTpl(e: FormEvent) {
    e.preventDefault();
    try {
      await updateMailTemplate(key, subject, body);
      setMsg(t("admin.mail_tpl_saved"));
    } catch (err) {
      setMsg(err instanceof Error ? err.message : t("admin.error"));
    }
  }

  async function saveSettings(e: FormEvent) {
    e.preventDefault();
    try {
      await putMailSettings({
        smtp: { host: smtpHost, port: smtpPort, user: smtpUser, password: smtpPass, from: smtpFrom },
        alerts: { auto_enabled: auto, auto_frequency: freq, global_threshold: threshold },
      });
      setMsg(t("admin.mail_settings_saved"));
    } catch (err) {
      setMsg(err instanceof Error ? err.message : t("admin.error"));
    }
  }

  return (
    <div className="admin-panel">
      <h2>{t("admin.tab_mail")}</h2>
      <p className="muted">
        {t("admin.mail_subscribers").replace("{count}", String(subs))} — {t("admin.mail_no_list")}
      </p>
      <form className="admin-form" onSubmit={(e) => void saveSettings(e)}>
        <h3>SMTP</h3>
        <input placeholder="host" value={smtpHost} onChange={(e) => setSmtpHost(e.target.value)} />
        <input placeholder="port" value={smtpPort} onChange={(e) => setSmtpPort(e.target.value)} />
        <input placeholder="user" value={smtpUser} onChange={(e) => setSmtpUser(e.target.value)} />
        <input
          placeholder="password"
          type="password"
          value={smtpPass}
          onChange={(e) => setSmtpPass(e.target.value)}
        />
        <input placeholder="from" value={smtpFrom} onChange={(e) => setSmtpFrom(e.target.value)} />
        <label>
          <input type="checkbox" checked={auto} onChange={(e) => setAuto(e.target.checked)} />{" "}
          {t("admin.mail_auto")}
        </label>
        <select value={freq} onChange={(e) => setFreq(e.target.value)}>
          <option value="hourly">hourly</option>
          <option value="daily">daily</option>
          <option value="weekly">weekly</option>
        </select>
        <input
          placeholder="threshold"
          value={threshold}
          onChange={(e) => setThreshold(e.target.value)}
        />
        <button type="submit" className="btn-primary">
          {t("admin.save")}
        </button>
        <button
          type="button"
          className="btn-secondary"
          onClick={() =>
            void sendShortageMail()
              .then((r) =>
                setMsg(
                  r.skipped_reason
                    ? t("admin.mail_skipped")
                    : t("admin.mail_sent").replace("{n}", String(r.sent)),
                ),
              )
              .catch((err) => setMsg(err instanceof Error ? err.message : t("admin.error")))
          }
        >
          {t("admin.mail_send_now")}
        </button>
      </form>
      <form className="admin-form" onSubmit={(e) => void saveTpl(e)}>
        <h3>{t("admin.mail_templates")}</h3>
        <select
          value={key}
          onChange={(e) => {
            const k = e.target.value;
            setKey(k);
            const cur = templates.find((x) => x.key === k);
            if (cur) {
              setSubject(cur.subject);
              setBody(cur.body_html);
            }
          }}
        >
          {templates.map((tpl) => (
            <option key={tpl.key} value={tpl.key}>
              {tpl.name}
            </option>
          ))}
        </select>
        <p className="muted">
          {t("admin.mail_tags")}:{" "}
          {(templates.find((x) => x.key === key)?.required_tags ?? []).join(", ")}
        </p>
        <input value={subject} onChange={(e) => setSubject(e.target.value)} />
        <textarea rows={8} value={body} onChange={(e) => setBody(e.target.value)} />
        <button type="submit" className="btn-primary">
          {t("admin.save")}
        </button>
      </form>
      {msg && <p className="ok-text">{msg}</p>}
    </div>
  );
}

export function SocialAdminPanel() {
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [kind, setKind] = useState("article");
  const [pending, setPending] = useState<
    Array<{ id: number; post_id: number; author_name: string; body: string }>
  >([]);
  const [msg, setMsg] = useState<string | null>(null);

  async function reload() {
    const r = await listSocialAdmin();
    setPending(r.pending_comments ?? []);
  }

  useEffect(() => {
    void reload().catch(() => setMsg(t("admin.error")));
  }, []);

  return (
    <div className="admin-panel">
      <h2>{t("admin.tab_social")}</h2>
      <form
        className="admin-form"
        onSubmit={(e) => {
          e.preventDefault();
          void createSocialPost({ kind, title, body, published: true })
            .then(() => {
              setTitle("");
              setBody("");
              setMsg(t("admin.social_published"));
            })
            .catch((err) => setMsg(err instanceof Error ? err.message : t("admin.error")));
        }}
      >
        <select value={kind} onChange={(e) => setKind(e.target.value)}>
          <option value="article">article</option>
          <option value="testimonial">testimonial</option>
        </select>
        <input value={title} onChange={(e) => setTitle(e.target.value)} placeholder="title" required />
        <textarea rows={5} value={body} onChange={(e) => setBody(e.target.value)} required />
        <button type="submit" className="btn-primary">
          {t("admin.publish")}
        </button>
      </form>
      <h3>{t("admin.social_moderate")}</h3>
      <ul>
        {pending.map((c) => (
          <li key={c.id}>
            <strong>{c.author_name}</strong>: {c.body}
            <div className="btn-row">
              <button
                type="button"
                className="btn-primary"
                onClick={() =>
                  void moderateComment(c.id, "approved").then(() => void reload())
                }
              >
                OK
              </button>
              <button
                type="button"
                className="btn-secondary"
                onClick={() =>
                  void moderateComment(c.id, "rejected").then(() => void reload())
                }
              >
                X
              </button>
            </div>
          </li>
        ))}
      </ul>
      {msg && <p className="ok-text">{msg}</p>}
    </div>
  );
}

export function UsersPanel() {
  const [users, setUsers] = useState<
    Array<{
      id: number;
      username: string;
      email?: string | null;
      display_name: string;
      is_bootstrap_admin: boolean;
      permissions: string[];
      active: boolean;
    }>
  >([]);
  const [allPerms, setAllPerms] = useState<string[]>([]);
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [display, setDisplay] = useState("");
  const [password, setPassword] = useState("");
  const [perms, setPerms] = useState<string[]>([]);
  const [msg, setMsg] = useState<string | null>(null);

  async function reload() {
    const r = await listUsers();
    setUsers(r.users ?? []);
    setAllPerms(r.all_permissions ?? []);
  }

  useEffect(() => {
    void reload().catch(() => setMsg(t("admin.error")));
  }, []);

  return (
    <div className="admin-panel">
      <h2>{t("admin.tab_users")}</h2>
      <ul>
        {users.map((u) => (
          <li key={u.id}>
            {u.username} {u.email ? `<${u.email}>` : "(bootstrap)"} —{" "}
            {(u.permissions ?? []).join(", ") || (u.is_bootstrap_admin ? "*" : "")}
            {!u.is_bootstrap_admin && (
              <button
                type="button"
                className="btn-secondary"
                onClick={() =>
                  void updateUser(u.id, {
                    permissions: u.permissions,
                    active: !u.active,
                  }).then(() => void reload())
                }
              >
                {u.active ? t("admin.deactivate") : t("admin.activate")}
              </button>
            )}
          </li>
        ))}
      </ul>
      <form
        className="admin-form"
        onSubmit={(e) => {
          e.preventDefault();
          void createUser({
            username,
            email,
            display_name: display,
            password,
            permissions: perms,
          })
            .then(() => {
              setUsername("");
              setEmail("");
              setDisplay("");
              setPassword("");
              setPerms([]);
              setMsg(t("admin.user_created"));
              return reload();
            })
            .catch((err) => setMsg(err instanceof Error ? err.message : t("admin.error")));
        }}
      >
        <h3>{t("admin.user_new")}</h3>
        <input
          placeholder="username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          required
        />
        <input
          placeholder="email"
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
        />
        <input
          placeholder="display"
          value={display}
          onChange={(e) => setDisplay(e.target.value)}
        />
        <input
          placeholder="password"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
        />
        <div className="perm-grid">
          {allPerms.map((p) => (
            <label key={p}>
              <input
                type="checkbox"
                checked={perms.includes(p)}
                onChange={(e) =>
                  setPerms(e.target.checked ? [...perms, p] : perms.filter((x) => x !== p))
                }
              />{" "}
              {p}
            </label>
          ))}
        </div>
        <button type="submit" className="btn-primary">
          {t("admin.create")}
        </button>
      </form>
      {msg && <p className="ok-text">{msg}</p>}
    </div>
  );
}

import { useEffect, useState, type FormEvent } from "react";
import { t } from "../i18n";
import {
  commentSocialPost,
  getSocialPostPublic,
  listSocialPostsPublic,
  type SocialPost,
} from "../services/api";

type Props = { onBack: () => void };

export function SocialPage({ onBack }: Props) {
  const [posts, setPosts] = useState<SocialPost[]>([]);
  const [selected, setSelected] = useState<SocialPost | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [author, setAuthor] = useState("");
  const [body, setBody] = useState("");
  const [msg, setMsg] = useState<string | null>(null);

  useEffect(() => {
    void listSocialPostsPublic()
      .then((r) => setPosts(r.posts ?? []))
      .catch(() => setError(t("social.error")));
  }, []);

  async function openPost(id: number) {
    setMsg(null);
    try {
      const p = await getSocialPostPublic(id);
      setSelected(p);
    } catch {
      setError(t("social.error"));
    }
  }

  async function onComment(e: FormEvent) {
    e.preventDefault();
    if (!selected) return;
    try {
      await commentSocialPost(selected.id, author.trim(), body.trim());
      setMsg(t("social.comment_queued"));
      setAuthor("");
      setBody("");
    } catch {
      setMsg(t("social.comment_error"));
    }
  }

  return (
    <div className="page admin-page">
      <header className="admin-top">
        <button type="button" className="btn-secondary" onClick={onBack}>
          {t("social.back")}
        </button>
        <h1>{t("social.title")}</h1>
      </header>
      <main className="content">
        {error && <p className="error-text">{error}</p>}
        {!selected && (
          <ul className="social-list">
            {posts.map((p) => (
              <li key={p.id}>
                <button type="button" className="social-card" onClick={() => void openPost(p.id)}>
                  <span className="muted">{p.kind}</span>
                  <strong>{p.title}</strong>
                </button>
              </li>
            ))}
            {posts.length === 0 && <p className="muted">{t("social.empty")}</p>}
          </ul>
        )}
        {selected && (
          <article className="social-article">
            <button type="button" className="btn-secondary" onClick={() => setSelected(null)}>
              {t("social.back_list")}
            </button>
            <h2>{selected.title}</h2>
            {selected.image_media_id && (
              <img
                src={`/api/v1/public/media/${selected.image_media_id}`}
                alt=""
                className="social-image"
              />
            )}
            <div className="social-body">{selected.body}</div>
            <h3>{t("social.comments")}</h3>
            <ul className="comment-list">
              {(selected.comments ?? []).map((c) => (
                <li key={c.id}>
                  <strong>{c.author_name}</strong>
                  <p>{c.body}</p>
                </li>
              ))}
            </ul>
            <form className="promise-form" onSubmit={(e) => void onComment(e)}>
              <label>
                <span>{t("social.author")}</span>
                <input value={author} onChange={(e) => setAuthor(e.target.value)} required />
              </label>
              <label>
                <span>{t("social.comment")}</span>
                <textarea value={body} onChange={(e) => setBody(e.target.value)} required rows={3} />
              </label>
              <button type="submit" className="btn-primary">
                {t("social.send_comment")}
              </button>
              {msg && <p className="ok-text">{msg}</p>}
            </form>
          </article>
        )}
      </main>
    </div>
  );
}

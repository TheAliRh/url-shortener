import { useCallback, useEffect, useState } from "react";

const API = import.meta.env.VITE_API_URL ?? "";
const SHORT_BASE =
  import.meta.env.VITE_SHORT_BASE ?? `${window.location.origin}/`;
const fmtDate = new Intl.DateTimeFormat(undefined, { dateStyle: "medium" });

async function api(path, { method = "GET", body, token } = {}) {
  const res = await fetch(API + path, {
    method,
    headers: {
      ...(body && { "Content-Type": "application/json" }),
      ...(token && { Authorization: `Bearer ${token}` }),
    },
    body: body && JSON.stringify(body),
  });
  const data = res.status === 204 ? null : await res.json().catch(() => null);
  if (!res.ok) {
    throw Object.assign(new Error(data?.error ?? "Something went wrong. Try again."), {
      status: res.status,
    });
  }
  return data;
}

function loadSession() {
  try { return JSON.parse(localStorage.getItem("session")); } catch { return null; }
}
function saveSession(s) {
  try {
    if (s) localStorage.setItem("session", JSON.stringify(s));
    else localStorage.removeItem("session");
  } catch { /* storage unavailable */ }
}

export default function App() {
  const [session, setSession] = useState(loadSession);

  const update = useCallback((s) => { saveSession(s); setSession(s); }, []);
  const expire = useCallback(() => update(null), [update]);

  async function signOut() {
    try { await api("/api/auth/logout", { method: "POST", token: session.token }); } catch { /* already gone */ }
    expire();
  }

  return (
    <div className="wrap">
      <header>
        <span className="brand">shrt</span>
        {session && (
          <div className="who">
            <span>{session.email}</span>
            <button className="ghost" onClick={signOut}>Sign out</button>
          </div>
        )}
      </header>
      {session ? <Dashboard session={session} onExpire={expire} /> : <AuthForm onAuth={update} />}
    </div>
  );
}

function AuthForm({ onAuth }) {
  const [mode, setMode] = useState("login"); // matches /api/auth/<mode>
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const signup = mode === "signup";

  async function submit(e) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const d = await api(`/api/auth/${mode}`, { method: "POST", body: { email, password } });
      onAuth({ token: d.token, email: d.user.email });
    } catch (err) {
      setError(err.message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <main>
      <h1>{signup ? "Create your account." : "Welcome back."}</h1>
      <form className="stack" onSubmit={submit}>
        <input type="email" aria-label="Email" placeholder="Email" autoComplete="email"
          value={email} onChange={(e) => setEmail(e.target.value)} required />
        <input type="password" aria-label="Password" placeholder="Password"
          autoComplete={signup ? "new-password" : "current-password"}
          value={password} onChange={(e) => setPassword(e.target.value)}
          minLength={signup ? 10 : undefined} required />
        {signup && <p className="hint">Use at least 10 characters.</p>}
        <button className="primary" disabled={busy}>
          {busy ? "One moment…" : signup ? "Create account" : "Sign in"}
        </button>
      </form>
      {error && <p className="error" role="alert">{error}</p>}
      <p className="switch">
        {signup ? "Already have an account? " : "New here? "}
        <button className="link" onClick={() => { setMode(signup ? "login" : "signup"); setError(""); }}>
          {signup ? "Sign in" : "Create an account"}
        </button>
      </p>
    </main>
  );
}

function Dashboard({ session, onExpire }) {
  const [url, setUrl] = useState("");
  const [links, setLinks] = useState(null);
  const [fresh, setFresh] = useState(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const call = useCallback(async (path, opts) => {
    try {
      return await api(path, { ...opts, token: session.token });
    } catch (err) {
      if (err.status === 401) onExpire();
      throw err;
    }
  }, [session.token, onExpire]);

  const load = useCallback(async () => {
    try {
      const d = await call("/api/url/");
      setLinks([...(d.urls ?? [])].sort((a, b) => new Date(b.created_at) - new Date(a.created_at)));
    } catch (err) {
      setError(err.message);
      setLinks([]);
    }
  }, [call]);

  useEffect(() => { load(); }, [load]);

  async function shorten(e) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const d = await call("/api/url/shorten", { method: "POST", body: { long_url: url.trim() } });
      setFresh(d.short_url);
      setUrl("");
      load();
    } catch (err) {
      setError(err.message);
    } finally {
      setBusy(false);
    }
  }

  async function remove(code) {
    try {
      await call(`/api/url/shortened/${code}`, { method: "DELETE" });
      setLinks((l) => l.filter((x) => x.short_url !== code));
      if (fresh?.endsWith(`/${code}`)) setFresh(null);
    } catch (err) {
      setError(err.message);
    }
  }

  return (
    <main>
      <h1>Make long links short.</h1>
      <form className="shorten" onSubmit={shorten}>
        <input type="url" aria-label="Long link" placeholder="Paste a long link, https://…"
          value={url} onChange={(e) => setUrl(e.target.value)} required />
        <button className="primary" disabled={busy}>{busy ? "Shortening…" : "Shorten"}</button>
      </form>
      {error && <p className="error" role="alert">{error}</p>}

      {fresh && (
        <div className="result">
          <a href={fresh} target="_blank" rel="noreferrer">{fresh.replace(/^https?:\/\//, "")}</a>
          <CopyButton text={fresh} className="primary" label="Copy link" />
        </div>
      )}

      <section className="list">
        <h2>Your links</h2>
        {links === null ? null : links.length === 0 ? (
          <p className="empty">No links yet. Paste one above to create your first.</p>
        ) : (
          <ul>
            {links.map((l) => <LinkRow key={l.short_url} link={l} onDelete={remove} />)}
          </ul>
        )}
      </section>
    </main>
  );
}

function LinkRow({ link, onDelete }) {
  const [confirm, setConfirm] = useState(false);
  const short = SHORT_BASE + link.short_url;

  return (
    <li className="row">
      <div className="main">
        <a className="code" href={short} target="_blank" rel="noreferrer">{link.short_url}</a>
        <div className="long" title={link.long_url}>{link.long_url}</div>
      </div>
      <span className="date">{fmtDate.format(new Date(link.created_at))}</span>
      <div className="actions">
        <CopyButton text={short} />
        {confirm ? (
          <button className="ghost danger" autoFocus onBlur={() => setConfirm(false)}
            onClick={() => onDelete(link.short_url)}>Confirm delete</button>
        ) : (
          <button className="ghost" onClick={() => setConfirm(true)}>Delete</button>
        )}
      </div>
    </li>
  );
}

function CopyButton({ text, className = "ghost", label = "Copy" }) {
  const [done, setDone] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(text);
      setDone(true);
      setTimeout(() => setDone(false), 1500);
    } catch { /* clipboard blocked */ }
  }

  return <button className={className} onClick={copy}>{done ? "Copied" : label}</button>;
}
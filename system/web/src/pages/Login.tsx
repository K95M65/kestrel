import { useState, useCallback, useEffect, useRef, type FormEvent } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { login } from "@/lib/api";
import { useTheme } from "@/lib/useTheme";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { C, PasswordField } from "@/components/setup/shared";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ReachyMark } from "@/components/ReachyMark";

// Login page — single password field that POSTs /api/login. A password query
// parameter supports one-click local-device links: it pre-fills the field and
// submits automatically. On success the server sets the os_session cookie
// (httpOnly + SameSite=Strict), and we navigate back to the page the user
// originally tried to reach (?next=…) or fall back to /monitor.
export default function Login() {
  const [theme, toggleTheme, themeClass] = useTheme();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  useDocumentTitle("Sign in");

  const passwordFromQuery = searchParams.get("password") ?? "";
  const [password, setPassword] = useState(passwordFromQuery);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const autoLoginAttempted = useRef(false);

  // `next` is captured from the URL so a bookmarked /edit lands the operator
  // back on /edit after login instead of always dumping them at /monitor.
  // Validated client-side: only same-origin pathnames are allowed (no
  // protocol-relative or absolute external URLs).
  const nextParam = searchParams.get("next") || "";
  const nextSafe =
    nextParam.startsWith("/") && !nextParam.startsWith("//") ? nextParam : "/monitor";

  const submitPassword = useCallback(async (value: string) => {
    if (!value) return;
    setError(null);
    setBusy(true);
    try {
      await login(value);
      // The app-level secret scrubber removes the query parameter. `next`
      // already includes the intended clean target URL and hash.
      navigate(nextSafe, { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setBusy(false);
    }
  }, [nextSafe, navigate]);

  const submit = useCallback((e: FormEvent) => {
    e.preventDefault();
    void submitPassword(password);
  }, [password, submitPassword]);

  useEffect(() => {
    if (!passwordFromQuery || autoLoginAttempted.current) return;
    autoLoginAttempted.current = true;
    void submitPassword(passwordFromQuery);
  }, [passwordFromQuery, submitPassword]);

  return (
    <div className={`lm-root ${themeClass}`} style={{
      minHeight: "100vh", display: "flex", alignItems: "center", justifyContent: "center",
      background: C.bg, color: C.text, fontSize: 14, padding: 24,
    }}>
      <Card className="w-full max-w-sm shadow-sm relative gap-4 py-5">
        <button onClick={toggleTheme} style={{
          position: "absolute", top: 12, right: 12,
          background: "none", border: "none", cursor: "pointer",
          fontSize: 14, color: C.textMuted, padding: "4px 6px",
        }} title={`Theme: ${theme}`} type="button">
          {theme === "dark" ? "◑" : "◐"}
        </button>
        <CardHeader className="items-center text-center pb-0">
          <ReachyMark size={48} />
          <div className="lm-reachy-brand-name" style={{ marginTop: 8 }}>Kestrel</div>
          <div className="lm-reachy-brand-sub">Desk Companion</div>
          <CardTitle className="text-xl mt-3">Sign in</CardTitle>
          <CardDescription className="text-left">
            Enter the admin password you set during device setup. If you haven't
            set one, the default is the 4 characters after the dash on the
            sticker at the bottom of your device.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {error && (
            <div style={{
              background: "var(--lm-red-dim)", border: "1px solid var(--lm-red-glow)",
              borderRadius: 8, padding: "9px 12px", fontSize: 12, color: C.red, marginBottom: 14,
            }}>
              {error}
            </div>
          )}
          <form onSubmit={submit}>
            <PasswordField
              label="Admin Password"
              id="login-password"
              value={password}
              onChange={setPassword}
              placeholder="••••••••"
            />
            <Button type="submit" className="w-full mt-1" disabled={busy || !password} size="lg">
              {busy ? "Signing in…" : "Sign in"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

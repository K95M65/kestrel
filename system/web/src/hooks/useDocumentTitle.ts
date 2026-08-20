import { useEffect } from "react";

const BASE = "Kestrel";

export function useDocumentTitle(parts: string | string[]) {
  // Derive the title during render so the effect depends on a single stable
  // string. `next` is a pure function of `parts`, so it changes exactly when
  // the old joined-parts dependency key would have — same re-run cadence, but
  // statically checkable by the linter.
  const segs = (Array.isArray(parts) ? parts : [parts]).filter(Boolean);
  const next = segs.length ? `${BASE} · ${segs.join(" · ")}` : BASE;
  useEffect(() => {
    const prev = document.title;
    document.title = next;
    return () => {
      document.title = prev;
    };
  }, [next]);
}

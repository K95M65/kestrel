import { type ReactNode } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { BookOpen } from "lucide-react";
import { isWikiSlug, wikiPage, WIKI_PAGES } from "@/lib/wiki";

function renderInline(line: string, keyPrefix: string, onOpen: (slug: string) => void): ReactNode[] {
  const parts: ReactNode[] = [];
  const re = /(\*\*(.+?)\*\*|\*(.+?)\*|`([^`]+)`|\[([^\]]+)\]\(([^)]+)\))/g;
  let last = 0;
  let match: RegExpExecArray | null;
  while ((match = re.exec(line)) !== null) {
    if (match.index > last) parts.push(line.slice(last, match.index));
    const k = `${keyPrefix}-${match.index}`;
    if (match[2]) parts.push(<strong key={k}>{match[2]}</strong>);
    else if (match[3]) parts.push(<em key={k}>{match[3]}</em>);
    else if (match[4]) parts.push(<code key={k} className="lm-wiki-code">{match[4]}</code>);
    else if (match[5] && match[6]) {
      const href = match[6].trim();
      if (isWikiSlug(href)) {
        parts.push(
          <button key={k} type="button" className="lm-wiki-link" onClick={() => onOpen(href)}>
            {match[5]}
          </button>,
        );
      } else {
        parts.push(
          <a key={k} href={href} target="_blank" rel="noopener noreferrer" className="lm-wiki-link">
            {match[5]}
          </a>,
        );
      }
    }
    last = match.index + match[0].length;
  }
  if (last < line.length) parts.push(line.slice(last));
  return parts.length ? parts : [line];
}

function WikiBody({ text, onOpen }: { text: string; onOpen: (slug: string) => void }) {
  const lines = text.replace(/\r\n/g, "\n").split("\n");
  const out: ReactNode[] = [];
  let i = 0;
  while (i < lines.length) {
    const line = lines[i];
    if (!line.trim()) { i++; continue; }
    if (line.startsWith("# ")) {
      out.push(<h1 key={`h-${i}`} className="lm-wiki-h1">{line.slice(2)}</h1>);
      i++; continue;
    }
    if (line.startsWith("## ")) {
      out.push(<h2 key={`h-${i}`} className="lm-wiki-h2">{line.slice(3)}</h2>);
      i++; continue;
    }
    if (/^[-*] /.test(line)) {
      const items: ReactNode[] = [];
      while (i < lines.length && /^[-*] /.test(lines[i])) {
        items.push(<li key={`li-${i}`}>{renderInline(lines[i].replace(/^[-*] /, ""), `li-${i}`, onOpen)}</li>);
        i++;
      }
      out.push(<ul key={`ul-${i}`} className="lm-wiki-ul">{items}</ul>);
      continue;
    }
    if (/^\d+\. /.test(line)) {
      const items: ReactNode[] = [];
      while (i < lines.length && /^\d+\. /.test(lines[i])) {
        items.push(<li key={`oli-${i}`}>{renderInline(lines[i].replace(/^\d+\. /, ""), `ol-${i}`, onOpen)}</li>);
        i++;
      }
      out.push(<ol key={`ol-${i}`} className="lm-wiki-ol">{items}</ol>);
      continue;
    }
    const para: string[] = [line];
    i++;
    while (i < lines.length && lines[i].trim() && !/^#{1,2} /.test(lines[i]) && !/^[-*] /.test(lines[i]) && !/^\d+\. /.test(lines[i])) {
      para.push(lines[i]);
      i++;
    }
    out.push(<p key={`p-${i}`} className="lm-wiki-p">{renderInline(para.join(" "), `p-${i}`, onOpen)}</p>);
  }
  return <>{out}</>;
}

export function GuideSection() {
  const loc = useLocation();
  const navigate = useNavigate();
  const doc = new URLSearchParams(loc.search).get("doc");
  const page = wikiPage(doc);

  function open(slug: string) {
    const params = new URLSearchParams(loc.search);
    if (!slug || slug === "index") params.delete("doc");
    else params.set("doc", slug);
    const qs = params.toString();
    navigate(`${loc.pathname}${qs ? `?${qs}` : ""}#guide`, { replace: true });
  }

  return (
    <div className="lm-wiki">
      <aside className="lm-wiki-nav" aria-label="Guide pages">
        <div className="lm-wiki-kicker">
          <BookOpen size={14} strokeWidth={1.9} />
          Guide
        </div>
        {WIKI_PAGES.map((p) => (
          <button
            key={p.slug}
            type="button"
            className={"lm-wiki-nav-item" + (p.slug === page.slug ? " lm-wiki-nav-item--on" : "")}
            onClick={() => open(p.slug)}
          >
            {p.title}
          </button>
        ))}
      </aside>
      <article className="lm-wiki-article">
        <WikiBody text={page.body} onOpen={open} />
        <p className="lm-wiki-foot">Same pages in the repo: docs/wiki</p>
      </article>
    </div>
  );
}

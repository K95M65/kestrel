/**
 * Tiny inline SVG companion. CSS drives idle motion (antenna bob, blink,
 * breathe) so we never load three.js / GSAP on the robot's browser.
 * Pauses under prefers-reduced-motion and when the tab is hidden (CSS
 * animation-play-state via .lm-root[data-hidden]).
 *
 * The tab icon is the static twin at public/favicon.svg — same viewBox and
 * geometry. If you change this mark, update the favicon too.
 */
export function ReachyMark({ size = 36, title = "Kestrel" }: { size?: number; title?: string }) {
  return (
    <svg
      className="lm-reachy"
      width={size}
      height={size}
      viewBox="0 0 64 64"
      role="img"
      aria-label={title}
    >
      <title>{title}</title>
      <g transform="translate(18 16)">
        <g className="lm-reachy-ant-l">
          <line x1="0" y1="0" x2="0" y2="-10" />
          <circle cx="0" cy="-10" r="3.2" />
        </g>
      </g>
      <g transform="translate(46 16)">
        <g className="lm-reachy-ant-r">
          <line x1="0" y1="0" x2="0" y2="-10" />
          <circle cx="0" cy="-10" r="3.2" />
        </g>
      </g>
      <g className="lm-reachy-body">
        <rect x="12" y="16" width="40" height="36" rx="14" />
        <circle className="lm-reachy-eye" cx="24" cy="32" r="4.2" />
        <circle className="lm-reachy-eye" cx="40" cy="32" r="4.2" />
        <path className="lm-reachy-smile" d="M26 42c3 3.4 9 3.4 12 0" />
      </g>
    </svg>
  );
}
